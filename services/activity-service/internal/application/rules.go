package application

import (
	lottery "activity/services/activity-service/internal/lottery/domain"
	"fmt"
)

type Rules struct {
	Pools  map[string]lottery.Pool `json:"pools"`
	Stages map[int]lottery.Prize   `json:"stages"`
	Chests []lottery.Prize         `json:"chests"`
}

func DefaultRules() Rules {
	choice := lottery.Prize{Code: "TRUE_LOVE_CHOICE", Name: "真爱无敌戒指选择权", Quantity: 1, Choices: []lottery.Prize{{Code: "1207751", ID: 1207751, Name: "真爱无敌戒指经典款", Quantity: 1}, {Code: "1207752", ID: 1207752, Name: "真爱无敌戒指铭文款", Quantity: 1}, {Code: "1207753", ID: 1207753, Name: "爱你万年戒指铭文款", Quantity: 1}}}
	gold := func(q, w int64) lottery.Prize {
		return lottery.Prize{Code: "1001", ID: 1001, Name: "金币", Quantity: q, Weight: w}
	}
	petal := func(q, w int64) lottery.Prize {
		return lottery.Prize{Code: "1002", ID: 1002, Name: "花瓣", Quantity: q, Weight: w}
	}
	ring := func(id uint64, name string, w int64) lottery.Prize {
		return lottery.Prize{Code: fmt.Sprint(id), ID: id, Name: name, Quantity: 1, Weight: w}
	}
	a := ring(1207244, "烟花之恋戒指", 100)
	b := ring(1203743, "闪闪心蝶戒指", 70)
	c := ring(1205251, "月亮游记戒指", 25)
	d := ring(1205470, "爱意翩跹戒指", 70)
	dayChoice := choice
	dayChoice.Weight = 5
	nightChoice := choice
	nightChoice.Weight = 30
	day := []lottery.Prize{gold(100, 350), gold(300, 250), petal(1, 200), a, b, c, dayChoice}
	a.Weight = 200
	b.Weight = 150
	c.Weight = 100
	night := []lottery.Prize{gold(1000, 250), petal(10, 200), a, b, c, d, nightChoice}
	threshold := []int64{300, 900, 3500, 10000, 30000, 53198, 56000, 150000, 250000, 350000, 500000, 710399, 800000, 980000, 1170000, 1367702, 1672100, 2082499}
	dayProb := []int64{1000000, 25000, 5770, 2310, 750, 650, 5350, 160, 150, 150, 100, 70, 170, 80, 80, 80, 50, 40}
	nightProb := []int64{1000000, 1000000, 1000000, 46150, 15000, 12930, 107060, 3190, 3000, 3000, 2000, 1430, 3350, 1670, 1580, 1520, 990, 730}
	dr, nr := make([]lottery.FlowerRule, 18), make([]lottery.FlowerRule, 18)
	for i := range dr {
		dr[i] = lottery.FlowerRule{Probability: dayProb[i], Guarantee: threshold[i]}
		nr[i] = lottery.FlowerRule{Probability: nightProb[i], Guarantee: threshold[i]}
	}
	return Rules{Pools: map[string]lottery.Pool{"DAY": {Cost: 5, Prizes: day, Flowers: dr}, "NIGHT": {Cost: 100, Prizes: night, Flowers: nr}}, Stages: map[int]lottery.Prize{3: gold(500, 0), 5: petal(10, 0), 8: a, 11: gold(2000, 0), 15: c, 17: choice}, Chests: []lottery.Prize{c, d, choice}}
}
