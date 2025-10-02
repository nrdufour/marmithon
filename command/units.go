package command

import (
	"fmt"
	"strconv"
	"strings"

	u "github.com/bcicen/go-units"
	hbot "github.com/whyrusleeping/hellabot"
)

func (core Core) ShowKnownUnits(m *hbot.Message) {
	core.Bot.Reply(m, "🔧 Unités disponibles (exemples):")
	core.Bot.Reply(m, "📏 Distance: m, km, ft, mi, in, cm, mm")
	core.Bot.Reply(m, "⚖️  Poids: kg, g, lb, oz, ton")
	core.Bot.Reply(m, "🌡️  Température: C, F, K")
	core.Bot.Reply(m, "📦 Volume: l, ml, gal, qt, pt")
	core.Bot.Reply(m, "💾 Données: B, KB, MB, GB, TB")
	core.Bot.Reply(m, "⚡ Énergie: J, kJ, cal, kcal, Wh, kWh")
	core.Bot.Reply(m, "💡 Usage: !convert <valeur> <unité_source> <unité_cible>")
	core.Bot.Reply(m, "🔍 Recherche: !convert search <terme> pour trouver des unités")
}

func (core Core) ConvertUnits(m *hbot.Message, args []string) {
	// Handle search functionality
	if len(args) >= 2 && strings.ToLower(args[0]) == "search" {
		core.SearchUnits(m, strings.Join(args[1:], " "))
		return
	}

	// Show help if wrong number of arguments
	if len(args) != 3 {
		core.ShowKnownUnits(m)
		return
	}

	// Parse the input value
	value, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		core.Bot.Reply(m, "❌ Valeur invalide. Utilisez un nombre (ex: 25.5)")
		return
	}

	unitFrom := strings.TrimSpace(args[1])
	unitTo := strings.TrimSpace(args[2])

	// Find the units
	fromUnit, err := u.Find(unitFrom)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("❌ Unité source inconnue: '%s'", unitFrom))
		core.suggestSimilarUnits(m, unitFrom)
		return
	}

	toUnit, err := u.Find(unitTo)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("❌ Unité cible inconnue: '%s'", unitTo))
		core.suggestSimilarUnits(m, unitTo)
		return
	}

	// Attempt conversion using the go-units library
	resultValue, err := u.ConvertFloat(value, fromUnit, toUnit)
	if err != nil {
		// Try to provide helpful error messages
		core.handleConversionError(m, err, unitFrom, unitTo)
		return
	}

	// Extract the float value from the result
	result := resultValue.Float()
	// Format the result with appropriate precision
	precision := determinePrecision(result)
	core.Bot.Reply(m, fmt.Sprintf("✅ %.2f %s = %.*f %s", value, unitFrom, precision, result, unitTo))
}

// SearchUnits helps users find available units
func (core Core) SearchUnits(m *hbot.Message, searchTerm string) {
	searchTerm = strings.ToLower(searchTerm)
	allUnits := u.All()
	var matches []u.Unit

	// Simple search through all units
	for _, unit := range allUnits {
		if strings.Contains(strings.ToLower(unit.Name), searchTerm) ||
			strings.Contains(strings.ToLower(unit.Symbol), searchTerm) {
			matches = append(matches, unit)
		}
	}

	if len(matches) == 0 {
		core.Bot.Reply(m, fmt.Sprintf("❌ Aucune unité trouvée pour '%s'", searchTerm))
		core.Bot.Reply(m, "💡 Essayez: meter, gram, celsius, liter, joule, byte")
		return
	}

	if len(matches) > 10 {
		matches = matches[:10]
	}

	core.Bot.Reply(m, fmt.Sprintf("🔍 Unités trouvées pour '%s':", searchTerm))
	for _, unit := range matches {
		core.Bot.Reply(m, fmt.Sprintf("  • %s (%s)", unit.Name, unit.Symbol))
	}
}

// handleConversionError provides helpful error messages
func (core Core) handleConversionError(m *hbot.Message, err error, unitFrom, unitTo string) {
	errMsg := err.Error()

	if strings.Contains(errMsg, "no conversion") || strings.Contains(errMsg, "incompatible") {
		core.Bot.Reply(m, fmt.Sprintf("❌ Impossible de convertir %s vers %s (types incompatibles)", unitFrom, unitTo))
		core.Bot.Reply(m, "💡 Vérifiez que vous convertissez entre le même type d'unité (ex: longueur vers longueur)")
	} else {
		core.Bot.Reply(m, fmt.Sprintf("❌ Erreur de conversion: %s", errMsg))
	}
}

// suggestSimilarUnits tries to find similar units to help the user
func (core Core) suggestSimilarUnits(m *hbot.Message, unitName string) {
	allUnits := u.All()
	var suggestions []u.Unit

	// Find units that start with the same letters or contain the term
	for _, unit := range allUnits {
		name := strings.ToLower(unit.Name)
		symbol := strings.ToLower(unit.Symbol)
		search := strings.ToLower(unitName)

		if strings.HasPrefix(name, search) || strings.HasPrefix(symbol, search) ||
			strings.Contains(name, search) || strings.Contains(symbol, search) {
			suggestions = append(suggestions, unit)
			if len(suggestions) >= 3 {
				break
			}
		}
	}

	if len(suggestions) > 0 {
		core.Bot.Reply(m, "💡 Unités similaires:")
		for _, unit := range suggestions {
			core.Bot.Reply(m, fmt.Sprintf("  • %s (%s)", unit.Name, unit.Symbol))
		}
	}
}

// determinePrecision chooses appropriate decimal places for the result
func determinePrecision(value float64) int {
	if value >= 1000 {
		return 0 // No decimals for large numbers
	} else if value >= 10 {
		return 1 // One decimal for medium numbers
	} else if value >= 1 {
		return 2 // Two decimals for numbers >= 1
	} else {
		return 4 // More decimals for small numbers
	}
}

// PerformConversion is a wrapper for external testing
func PerformConversion(value float64, unitOrigin, unitDest string) (float64, error) {
	fromUnit, err := u.Find(unitOrigin)
	if err != nil {
		return 0, err
	}
	toUnit, err := u.Find(unitDest)
	if err != nil {
		return 0, err
	}
	result, err := u.ConvertFloat(value, fromUnit, toUnit)
	if err != nil {
		return 0, err
	}
	return result.Float(), nil
}
