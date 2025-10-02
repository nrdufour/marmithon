package main

import (
	"fmt"
	"marmithon/command"
)

func main() {
	fmt.Println("Testing nautical mile support:")

	// Test km to nautical miles
	result, err := command.PerformConversion(1.852, "km", "nmi")
	if err != nil {
		fmt.Printf("❌ 1.852 km to nmi failed: %v\n", err)
	} else {
		fmt.Printf("✅ 1.852 km = %.3f nmi (should be ~1.000)\n", result)
	}

	// Test nautical miles to km
	result, err = command.PerformConversion(1, "nmi", "km")
	if err != nil {
		fmt.Printf("❌ 1 nmi to km failed: %v\n", err)
	} else {
		fmt.Printf("✅ 1 nmi = %.3f km (should be 1.852)\n", result)
	}

	// Test alternative name
	result, err = command.PerformConversion(1, "nautical", "km")
	if err != nil {
		fmt.Printf("❌ 1 nautical to km failed: %v\n", err)
	} else {
		fmt.Printf("✅ 1 nautical = %.3f km (should be 1.852)\n", result)
	}

	fmt.Println("✅ Nautical mile support restored!")
}