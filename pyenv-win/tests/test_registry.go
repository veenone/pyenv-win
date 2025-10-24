package main

import (
	"fmt"
	"os"

	"github.com/pyenv-win/pyenv-win/internal/registry"
)

func main() {
	testVersion := "3.8.10"
	testPath := `C:\Users\ARaha\.pyenv\pyenv-win\versions\3.8.10`

	fmt.Println("=== Testing Windows Registry Integration ===\n")

	// Test 1: List currently registered versions
	fmt.Println("1. Listing currently registered Python versions...")
	versions, err := registry.ListRegisteredVersions()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		if len(versions) == 0 {
			fmt.Println("   No versions currently registered")
		} else {
			fmt.Printf("   Found %d registered versions:\n", len(versions))
			for _, v := range versions {
				fmt.Printf("   - %s\n", v)
			}
		}
	}
	fmt.Println()

	// Test 2: Check if test version is registered
	fmt.Printf("2. Checking if %s is registered...\n", testVersion)
	isReg := registry.IsRegistered(testVersion)
	fmt.Printf("   Result: %v\n\n", isReg)

	// Test 3: Register the version
	fmt.Printf("3. Registering %s...\n", testVersion)
	err = registry.RegisterVersion(testVersion, testPath)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   Successfully registered!")
	fmt.Println()

	// Test 4: Verify it's now registered
	fmt.Printf("4. Verifying %s is now registered...\n", testVersion)
	isReg = registry.IsRegistered(testVersion)
	fmt.Printf("   Result: %v\n", isReg)
	if !isReg {
		fmt.Println("   ERROR: Version should be registered but isn't!")
		os.Exit(1)
	}
	fmt.Println()

	// Test 5: Get registered path
	fmt.Printf("5. Getting registered path for %s...\n", testVersion)
	regPath, err := registry.GetRegisteredPath(testVersion)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Path: %s\n", regPath)
	fmt.Println()

	// Test 6: List registered versions again
	fmt.Println("6. Listing registered versions again...")
	versions, err = registry.ListRegisteredVersions()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Found %d registered versions:\n", len(versions))
		for _, v := range versions {
			isOurs := ""
			if v == testVersion {
				isOurs = " (our test version)"
			}
			fmt.Printf("   - %s%s\n", v, isOurs)
		}
	}
	fmt.Println()

	// Test 7: Unregister the version
	fmt.Printf("7. Unregistering %s...\n", testVersion)
	err = registry.UnregisterVersion(testVersion)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   Successfully unregistered!")
	fmt.Println()

	// Test 8: Verify it's no longer registered
	fmt.Printf("8. Verifying %s is no longer registered...\n", testVersion)
	isReg = registry.IsRegistered(testVersion)
	fmt.Printf("   Result: %v\n", isReg)
	if isReg {
		fmt.Println("   ERROR: Version should not be registered but is!")
		os.Exit(1)
	}
	fmt.Println()

	fmt.Println("=== All registry tests passed! ===")
}
