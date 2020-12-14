package mandalorian

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"strconv"

	"github.com/NJUPT-ISL/Mandalorian/pkg/scheduler/framework"
	nodecontroller "github.com/NJUPT-ISL/NodeSimulator/pkg/controllers/node"
	scv "github.com/NJUPT-ISL/SCV/api/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func (m *Mandalorian) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	klog.Infof("Filter Node: %v while Scheduling Pod: %v/%v. ",nodeInfo.Node().GetName(),pod.GetNamespace(),pod.GetName())
	// TODO: Write Your Filter Policy here.
	// ..

	currentScv := &scv.Scv{}
	err := m.scvClient.Get(ctx, types.NamespacedName{Name: nodeInfo.Node().GetName()}, currentScv)
	if err != nil {
		klog.Errorf("Get SCV Error: %v", err)
		return framework.NewStatus(framework.Unschedulable, "Node:"+nodeInfo.Node().Name+" "+err.Error())
	}

	if pod.GetLabels()[nodecontroller.Affinity] != "" ||
		pod.GetLabels()[nodecontroller.AntiAffinity] != "" ||
		pod.GetLabels()[nodecontroller.Exclusion] != ""{
		if ok, _ := PodCheckAffinityTags(pod, currentScv); !ok {
			return framework.NewStatus(framework.Unschedulable, "Pod Affinity make" + nodeInfo.Node().Name + "unschedulable")
		}
	}

	if ok, number := PodFitsNumber(pod, currentScv); ok {
		isFitsMemory, _ := PodFitsMemory(number, pod, currentScv)
		isFitsClock, _ := PodFitsClock(number, pod, currentScv)
		if isFitsMemory && isFitsClock {
			return framework.NewStatus(framework.Success, "")
		}
	}
	return framework.NewStatus(framework.Unschedulable, "Node:"+nodeInfo.Node().Name)
	return nil
}

func PodCheckAffinityTags(pod *v1.Pod, scv1 *scv.Scv) (bool, scv.CardList) {
	scheduleFlag := false
	filterCardList := scv1.Status.CardList
	if pod.GetLabels()[nodecontroller.Affinity] != "" {
		isFound := false
		for _, card := range scv1.Status.CardList {
			for _, value := range card.AffinityTag {
				if pod.GetLabels()[nodecontroller.Affinity] != value {
					continue
				}
				isFound = true
			}

			if !isFound {
				RemoveParam(filterCardList, card)
			}

		}
		if !isFound {
			return scheduleFlag, nil
		}
	}

	if pod.GetLabels()[nodecontroller.AntiAffinity] != "" {
		isFound := false
		for _, card := range scv1.Status.CardList {
			for _, value := range card.AntiAffinityTag {
				if pod.GetLabels()[nodecontroller.AntiAffinity] == value {
					isFound = true
					RemoveParam(filterCardList, card)
					break
				}
			}
		}
		if isFound {
			return scheduleFlag, nil
		}
	}

	if pod.GetLabels()[nodecontroller.Exclusion] != "" {
		isFound := false
		for _, card := range scv1.Status.CardList {
			if len(card.ExclusionTag) != 0 {
				for _, value := range card.ExclusionTag {
					if pod.GetLabels()[nodecontroller.Exclusion] == value {
						isFound = true
					}
					if !isFound {
						RemoveParam(filterCardList, card)
					}
					break
				}
			}
		}
		if !isFound {
			return scheduleFlag, nil
		}
	}

	scheduleFlag = true
	return scheduleFlag, filterCardList

}

func PodFitsNumber(pod *v1.Pod, scv *scv.Scv) (bool, uint) {
	if number, ok := pod.GetLabels()["scv/number"]; ok {
		return strToUint(number) <= scv.Status.CardNumber, strToUint(number)
	}
	return scv.Status.CardNumber > 0, 1
}

func PodFitsMemory(number uint, pod *v1.Pod, scv *scv.Scv) (bool, uint64) {
	if memory, ok := pod.GetLabels()["scv/memory"]; ok {
		fitsCard := uint(0)
		m := StrToUint64(memory)
		for _, card := range scv.Status.CardList {
			if CardFitsMemory(m, card) {
				fitsCard++
			}
		}
		if fitsCard >= number {
			return true, m
		}
		return false, m
	}
	return true, 0
}

func PodFitsClock(number uint, pod *v1.Pod, scv *scv.Scv) (bool, uint) {
	if clock, ok := pod.GetLabels()["scv/clock"]; ok {
		fitsCard := uint(0)
		c := strToUint(clock)
		for _, card := range scv.Status.CardList {
			if CardFitsClock(c, card) {
				fitsCard++
			}
		}
		if fitsCard >= number {
			return true, c
		}
		return false, c
	}
	return true, 0
}

func CardFitsMemory(memory uint64, card scv.Card) bool {
	return card.Health == "Healthy" && card.FreeMemory >= memory
}

func CardFitsClock(clock uint, card scv.Card) bool {
	return card.Health == "Healthy" && card.Clock == clock
}

func strToUint(str string) uint {
	if i, e := strconv.Atoi(str); e != nil {
		return 0
	} else {
		return uint(i)
	}
}

func StrToUint64(str string) uint64 {
	if i, e := strconv.Atoi(str); e != nil {
		return 0
	} else {
		return uint64(i)
	}
}

func StrToInt64(str string) int64 {
	if i, e := strconv.Atoi(str); e != nil {
		return 0
	} else {
		return int64(i)
	}
}

func Uint64ToInt64(intNum uint64) int64 {
	return StrToInt64(strconv.FormatUint(intNum, 10))
}

func RemoveParam(sli []scv.Card, n scv.Card) []scv.Card {
	for i := 0; i < len(sli); i++ {
		if sli[i].ID == n.ID {
			if i == 0 {
				sli = sli[1:]
			} else if i == len(sli) - 1 {
				sli = sli[:i]
			} else {
				sli = append(sli[:i], sli[i+1:]...)
			}
			i--
		}
	}
	return sli
}
