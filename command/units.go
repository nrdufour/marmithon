package command

import (
	"fmt"
	"strconv"

	hbot "github.com/whyrusleeping/hellabot"
)

type UnitType int

const (
	Distance UnitType = iota
	Weight
	Volume
)

type Unit struct {
	Name   string
	Abbrev string
	Type   UnitType
}

var KnownUnits = map[string]Unit{
	"m":  {"mètre", "m", Distance},
	"ft": {"feet", "ft", Distance},
}

type Conversion struct {
	Factor float64
	// maybe more later on
}

var ConversionTable = map[string]Conversion{
	"ft/m": {0.3048},
}

func (core Core) ShowKnownUnits(m *hbot.Message) {
	core.Bot.Reply(m, "Unités connues:")
	for _, unit := range KnownUnits {
		core.Bot.Reply(m, fmt.Sprintf(">> '%s'/'%s", unit.Name, unit.Abbrev))
	}
}

func (core Core) ConvertUnits(m *hbot.Message, args []string) {
	if len(args) != 3 {
		core.ShowKnownUnits(m)
		return
	}

	value, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("Désolé, je ne comprends pas"))
		return
	}
	unitOriginRaw := args[1]
	unitDestRaw := args[2]

	var conversionId string
	var reverseConversion = false
	if unitOriginRaw < unitDestRaw {
		conversionId = fmt.Sprintf("%s/%s", unitOriginRaw, unitDestRaw)
	} else {
		conversionId = fmt.Sprintf("%s/%s", unitDestRaw, unitOriginRaw)
		reverseConversion = true
	}

	conversion, exists := ConversionTable[conversionId]
	if exists {
		var newValue float64
		if reverseConversion {
			newValue = value / conversion.Factor
		} else {
			newValue = value * conversion.Factor
		}
		core.Bot.Reply(m, fmt.Sprintf("%f %s est égal à %f %s", value, unitOriginRaw, newValue, unitDestRaw))
	} else {
		core.Bot.Reply(m, fmt.Sprintf("Désolé, je ne connais pas encore cette conversion"))
	}
}
