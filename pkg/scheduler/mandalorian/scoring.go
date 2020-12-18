package mandalorian

import (
	"context"
	"errors"
	"fmt"

	"github.com/NJUPT-ISL/Mandalorian/pkg/scheduler/framework"
	nodecontroller "github.com/NJUPT-ISL/NodeSimulator/pkg/controllers/node"
	scv "github.com/NJUPT-ISL/SCV/api/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func (m *Mandalorian) Score(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.Infof("Scoring Node: %v while scheduling Pod: %v/%v", nodeName, pod.GetNamespace(), pod.GetName())
	// TODO: Write Your Score Policy here.
	// ...
	// Get Node Info
	nodeInfo, err := m.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v", nodeName, err))
	}

	// Get Scv Info
	currentScv := &scv.Scv{}
	err = m.scvClient.Get(ctx, types.NamespacedName{Name: nodeName}, currentScv)
	if err != nil {
		klog.Errorf("Get SCV Error: %v", err)
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("Score Node Error: %v", err))
	}

	uNodeScore, err := CalculateScoreByBestFit(currentScv, pod, *nodeInfo)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("Score Node Error: %v", err))
	}
	nodeScore := Uint64ToInt64(uNodeScore)
	return nodeScore, framework.NewStatus(framework.Success, "")
}

func (m *Mandalorian) NormalizeScore(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	var (
		highest int64 = 0
		lowest        = scores[0].Score
	)

	for _, nodeScore := range scores {
		if nodeScore.Score < lowest {
			lowest = nodeScore.Score
		}
		if nodeScore.Score > highest {
			highest = nodeScore.Score
		}
	}

	if highest == lowest {
		lowest--
	}

	// Set Range to [0-100]
	for i, nodeScore := range scores {
		scores[i].Score = (nodeScore.Score - lowest) * framework.MaxNodeScore / (highest - lowest)
		klog.Infof("Node: %v, Score: %v in Plugin: Mandalorian When scheduling Pod: %v/%v", scores[i].Name, scores[i].Score, pod.GetNamespace(), pod.GetName())
	}
	return nil
}

func (m *Mandalorian) ScoreExtensions() framework.ScoreExtensions {
	return m
}

func CalculateScoreByBestFit(scv *scv.Scv, pod *v1.Pod, nodeInfo framework.NodeInfo) (uint64, error) {
	var finalScore = uint64(0)

	scoreCardList := scv.Status.CardList

	type candidateNodeGPU struct {
		GPUID int
		Point uint64
	}

	bestNode := candidateNodeGPU{
		Point: ^uint64(0),
		GPUID: 0,
	}

	if pod.GetLabels()[nodecontroller.Affinity] != "" ||
		pod.GetLabels()[nodecontroller.AntiAffinity] != "" ||
		pod.GetLabels()[nodecontroller.Exclusion] != "" {
		ok, filterCardList := PodCheckAffinityTags(pod, scv)
		if !ok {
			return 0, errors.New(nodeInfo.Node().Name + "can not satisfy Pod Affinity Tag")
		}
		scoreCardList = filterCardList
	}

	mem := StrToUint64(pod.GetLabels()["scv/memory"])

	var cardScore = make([]uint64, len(scoreCardList))

	for index, card := range scoreCardList {
		sub := card.FreeMemory - mem
		if sub >= 0 && sub < bestNode.Point {
			bestNode.Point = sub
			bestNode.GPUID = index
		}
	}

	cardScore[bestNode.GPUID] = uint64(100000) - bestNode.Point

	for _, value := range cardScore {
		finalScore += value
	}

	return finalScore, nil

}
