package command

import (
	"fmt"
	"strconv"
	"strings"

	u "github.com/bcicen/go-units"
	hbot "github.com/whyrusleeping/hellabot"
)

// Custom units that go-units doesn't support
// Using a simpler approach without creating Unit objects to avoid conflicts
var customUnitNames = []string{"nmi", "nautical", "nauticalmile"}

// Custom conversions for units not in go-units
var customConversions = map[string]float64{
	"km_to_nmi": 1.0 / 1.852,    // 1 km = 1/1.852 nautical miles
	"nmi_to_km": 1.852,          // 1 nautical mile = 1.852 km
	"m_to_nmi":  1.0 / 1852.0,   // 1 meter = 1/1852 nautical miles
	"nmi_to_m":  1852.0,         // 1 nautical mile = 1852 meters
}

func (core Core) ShowKnownUnits(bot *hbot.Bot, m *hbot.Message) {
	bot.Reply(m, "🔧 Unités disponibles (exemples):")
	bot.Reply(m, "📏 Distance: m, km, ft, mi, in, cm, mm, nmi")
	bot.Reply(m, "⚖️  Poids: kg, g, lb, oz, ton")
	bot.Reply(m, "🌡️  Température: C, F, K")
	bot.Reply(m, "📦 Volume: l, ml, gal, qt, pt")
	bot.Reply(m, "💾 Données: B, KB, MB, GB, TB")
	bot.Reply(m, "⚡ Énergie: J, kJ, cal, kcal, Wh, kWh")
	bot.Reply(m, "💡 Usage: !convert <valeur> <unité_source> <unité_cible>")
	bot.Reply(m, "🔍 Recherche: !convert search <terme> pour trouver des unités")
}

func (core Core) ConvertUnits(bot *hbot.Bot, m *hbot.Message, args []string) {
	// Handle search functionality
	if len(args) >= 2 && strings.ToLower(args[0]) == "search" {
		core.SearchUnits(bot, m, strings.Join(args[1:], " "))
		return
	}

	// Show help if wrong number of arguments
	if len(args) != 3 {
		core.ShowKnownUnits(bot, m)
		return
	}

	// Parse the input value
	value, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		bot.Reply(m, "❌ Valeur invalide. Utilisez un nombre (ex: 25.5)")
		return
	}

	unitFrom := strings.TrimSpace(args[1])
	unitTo := strings.TrimSpace(args[2])

	// Try custom conversion first (for nautical miles, etc.)
	if result, ok := core.tryCustomConversion(value, unitFrom, unitTo); ok {
		precision := determinePrecision(result)
		bot.Reply(m, fmt.Sprintf("✅ %.2f %s = %.*f %s", value, unitFrom, precision, result, unitTo))
		return
	}

	// Find the units in the standard library
	fromUnit, err := u.Find(unitFrom)
	if err != nil {
		bot.Reply(m, fmt.Sprintf("❌ Unité source inconnue: '%s'", unitFrom))
		core.suggestSimilarUnits(bot, m, unitFrom)
		return
	}

	toUnit, err := u.Find(unitTo)
	if err != nil {
		bot.Reply(m, fmt.Sprintf("❌ Unité cible inconnue: '%s'", unitTo))
		core.suggestSimilarUnits(bot, m, unitTo)
		return
	}

	// Attempt conversion using the go-units library
	resultValue, err := u.ConvertFloat(value, fromUnit, toUnit)
	if err != nil {
		// Try to provide helpful error messages
		core.handleConversionError(bot, m, err, unitFrom, unitTo)
		return
	}

	// Extract the float value from the result
	result := resultValue.Float()
	// Format the result with appropriate precision
	precision := determinePrecision(result)
	bot.Reply(m, fmt.Sprintf("✅ %.2f %s = %.*f %s", value, unitFrom, precision, result, unitTo))
}

// tryCustomConversion handles conversions for custom units like nautical miles
func (core Core) tryCustomConversion(value float64, unitFrom, unitTo string) (float64, bool) {
	unitFrom = strings.ToLower(unitFrom)
	unitTo = strings.ToLower(unitTo)

	// Normalize unit names
	if unitFrom == "nautical" || unitFrom == "nauticalmile" {
		unitFrom = "nmi"
	}
	if unitTo == "nautical" || unitTo == "nauticalmile" {
		unitTo = "nmi"
	}

	// Check for direct custom conversions
	conversionKey := fmt.Sprintf("%s_to_%s", unitFrom, unitTo)
	if factor, exists := customConversions[conversionKey]; exists {
		return value * factor, true
	}

	// Check for reverse conversions
	reverseKey := fmt.Sprintf("%s_to_%s", unitTo, unitFrom)
	if factor, exists := customConversions[reverseKey]; exists {
		return value / factor, true
	}

	// Try intermediate conversions through meters for nautical miles
	if unitFrom == "nmi" {
		// nmi -> meters -> target
		meters := value * 1852.0
		if unitTo == "km" {
			return meters / 1000.0, true
		} else if unitTo == "m" || unitTo == "meter" {
			return meters, true
		}
	} else if unitTo == "nmi" {
		// source -> meters -> nmi
		var meters float64
		if unitFrom == "km" || unitFrom == "kilometer" {
			meters = value * 1000.0
		} else if unitFrom == "m" || unitFrom == "meter" {
			meters = value
		} else {
			return 0, false
		}
		return meters / 1852.0, true
	}

	return 0, false
}

// SearchUnits helps users find available units
func (core Core) SearchUnits(bot *hbot.Bot, m *hbot.Message, searchTerm string) {
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

	// Check for custom units separately
	var customMatches []string
	for _, name := range customUnitNames {
		if strings.Contains(strings.ToLower(name), searchTerm) ||
			strings.Contains("nautical mile", searchTerm) {
			customMatches = append(customMatches, name)
		}
	}

	if len(matches) == 0 && len(customMatches) == 0 {
		bot.Reply(m, fmt.Sprintf("❌ Aucune unité trouvée pour '%s'", searchTerm))
		bot.Reply(m, "💡 Essayez: meter, gram, celsius, liter, joule, byte, nmi")
		return
	}

	if len(matches) > 10 {
		matches = matches[:10]
	}

	bot.Reply(m, fmt.Sprintf("🔍 Unités trouvées pour '%s':", searchTerm))
	for _, unit := range matches {
		bot.Reply(m, fmt.Sprintf("  • %s (%s)", unit.Name, unit.Symbol))
	}
	for _, customUnit := range customMatches {
		bot.Reply(m, fmt.Sprintf("  • nautical mile (%s) - distance", customUnit))
	}
}

// handleConversionError provides helpful error messages
func (core Core) handleConversionError(bot *hbot.Bot, m *hbot.Message, err error, unitFrom, unitTo string) {
	errMsg := err.Error()

	if strings.Contains(errMsg, "no conversion") || strings.Contains(errMsg, "incompatible") {
		bot.Reply(m, fmt.Sprintf("❌ Impossible de convertir %s vers %s (types incompatibles)", unitFrom, unitTo))
		bot.Reply(m, "💡 Vérifiez que vous convertissez entre le même type d'unité (ex: longueur vers longueur)")
	} else {
		bot.Reply(m, fmt.Sprintf("❌ Erreur de conversion: %s", errMsg))
	}
}

// suggestSimilarUnits tries to find similar units to help the user
func (core Core) suggestSimilarUnits(bot *hbot.Bot, m *hbot.Message, unitName string) {
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
		bot.Reply(m, "💡 Unités similaires:")
		for _, unit := range suggestions {
			bot.Reply(m, fmt.Sprintf("  • %s (%s)", unit.Name, unit.Symbol))
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
	// Try custom conversion first
	core := Core{} // dummy core for method access
	if result, ok := core.tryCustomConversion(value, unitOrigin, unitDest); ok {
		return result, nil
	}

	// Fall back to standard library
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
