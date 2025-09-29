package command

import (
	"errors"
	"fmt"
	"strconv"

	hbot "github.com/whyrusleeping/hellabot"
)

type UnitType int

const (
	Distance UnitType = iota
	Weight
	Volume
	Temperature
)

type Unit struct {
	Name   string
	Abbrev string
	Type   UnitType
}

var KnownUnits = map[string]Unit{
	// Distance
	"m":  {"mètre", "m", Distance},
	"ft": {"feet", "ft", Distance},
	"km": {"kilomètre", "km", Distance},
	"nm": {"nautical mile", "nm", Distance},

	// Weight
	"kg": {"kilogramme", "kg", Weight},
	"lb": {"pound", "lb", Weight},

	// Volume
	"l":   {"litre", "l", Volume},
	"gal": {"gallon", "gal", Volume},

	// Temperature
	"c": {"Celsius", "°C", Temperature},
	"f": {"Fahrenheit", "°F", Temperature},
}

type Conversion struct {
	Factor float64
	Offset float64 // For temperature conversions
	// maybe more later on
}

var ConversionTable = map[string]Conversion{
	// Distance conversions
	"ft/m":  {0.3048, 0},
	"km/m":  {1000, 0},
	"km/nm": {1.852, 0},

	// Weight conversions
	"kg/lb": {2.20462, 0},

	// Volume conversions (US gallon)
	"gal/l": {3.78541, 0},

	// Temperature conversions (special handling needed)
	"c/f": {1.8, 32}, // F = C * 1.8 + 32
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
		core.Bot.Reply(m, "Désolé, je ne comprends pas")
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

		// Special handling for temperature conversions
		originUnit := KnownUnits[unitOriginRaw]
		if originUnit.Type == Temperature {
			if reverseConversion {
				// Converting from F to C: C = (F - 32) / 1.8
				newValue = (value - conversion.Offset) / conversion.Factor
			} else {
				// Converting from C to F: F = C * 1.8 + 32
				newValue = value*conversion.Factor + conversion.Offset
			}
		} else {
			// Standard linear conversion
			if reverseConversion {
				newValue = value / conversion.Factor
			} else {
				newValue = value * conversion.Factor
			}
		}

		core.Bot.Reply(m, fmt.Sprintf("%.2f %s est égal à %.2f %s", value, unitOriginRaw, newValue, unitDestRaw))
	} else {
		core.Bot.Reply(m, "Désolé, je ne connais pas encore cette conversion")
	}
}

func validateUnits(unitOrigin, unitDest string) error {
	originUnit, originExists := KnownUnits[unitOrigin]
	destUnit, destExists := KnownUnits[unitDest]

	if !originExists {
		return fmt.Errorf("unité d'origine inconnue: %s", unitOrigin)
	}
	if !destExists {
		return fmt.Errorf("unité de destination inconnue: %s", unitDest)
	}
	if originUnit.Type != destUnit.Type {
		return fmt.Errorf("impossible de convertir entre des types d'unités différents (%s vers %s)", unitOrigin, unitDest)
	}
	return nil
}

func performConversion(value float64, unitOrigin, unitDest string) (float64, error) {
	var conversionId string
	var reverseConversion bool

	if unitOrigin < unitDest {
		conversionId = fmt.Sprintf("%s/%s", unitOrigin, unitDest)
	} else {
		conversionId = fmt.Sprintf("%s/%s", unitDest, unitOrigin)
		reverseConversion = true
	}

	conversion, exists := ConversionTable[conversionId]
	if !exists {
		return 0, errors.New("désolé, je ne connais pas encore cette conversion")
	}

	// Special handling for temperature conversions
	originUnit := KnownUnits[unitOrigin]
	if originUnit.Type == Temperature {
		if reverseConversion {
			// Converting from F to C: C = (F - 32) / 1.8
			return (value - conversion.Offset) / conversion.Factor, nil
		} else {
			// Converting from C to F: F = C * 1.8 + 32
			return value*conversion.Factor + conversion.Offset, nil
		}
	}

	// Standard linear conversion
	if reverseConversion {
		return value / conversion.Factor, nil
	}
	return value * conversion.Factor, nil
}
