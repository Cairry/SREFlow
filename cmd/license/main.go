package main

import (
	"SREFlow/pkg/license"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	genKeyCmd := flag.NewFlagSet("genkey", flag.ExitOnError)
	genKeyPriv := genKeyCmd.String("private", "config/license.pem", "Output path for private key")

	genLicCmd := flag.NewFlagSet("genlic", flag.ExitOnError)
	licHolder := genLicCmd.String("holder", "", "License holder (company name), required")
	licDuration := genLicCmd.String("duration", "365d", "License validity duration (e.g. 30d, 365d, 0 for perpetual)")
	licMachineID := genLicCmd.String("machine-id", "", "Machine ID to bind to (optional)")
	licMaxWorkspaces := genLicCmd.Int("max-workspaces", 0, "Maximum number of workspaces (0 = unlimited)")
	licMaxWorkflows := genLicCmd.Int("max-workflows", 0, "Maximum number of workflows (0 = unlimited)")
	licMaxMCP := genLicCmd.Int("max-mcp", 0, "Maximum number of MCP servers (0 = unlimited)")
	licMaxSkills := genLicCmd.Int("max-skills", 0, "Maximum number of skills (0 = unlimited)")
	licPrivKey := genLicCmd.String("private-key", "config/license.pem", "Path to private key")
	licOutput := genLicCmd.String("output", "config/license.lic", "Output path for license file")

	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	verifyLicPath := verifyCmd.String("license", "config/license.lic", "Path to license file")

	if len(os.Args) < 2 {
		fmt.Println("SREFlow License Tool")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  genkey    Generate RSA key pair")
		fmt.Println("  genlic    Generate a signed license file")
		fmt.Println("  verify    Verify a license file")
		fmt.Println()
		fmt.Println("Use \\<command> -h\\> for more information about a command.")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "genkey":
		genKeyCmd.Parse(os.Args[2:])
		if err := license.GenerateKeyPair(*genKeyPriv); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Private key: %s\n", *genKeyPriv)

	case "genlic":
		genLicCmd.Parse(os.Args[2:])
		if *licHolder == "" {
			fmt.Fprintln(os.Stderr, "Error: --holder is required")
			genLicCmd.Usage()
			os.Exit(1)
		}

		duration, err := parseDuration(*licDuration)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		now := time.Now()
		lic := license.License{
			Holder:        *licHolder,
			IssuedAt:      now.Unix(),
			MaxWorkspaces: *licMaxWorkspaces,
			MaxWorkflows:  *licMaxWorkflows,
			MaxMCP:        *licMaxMCP,
			MaxSkills:     *licMaxSkills,
		}

		if duration > 0 {
			lic.ExpiresAt = now.Add(duration).Unix()
		}

		if *licMachineID != "" {
			lic.MachineID = *licMachineID
		}

		licFile, err := license.Sign(lic, *licPrivKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		data, err := json.MarshalIndent(licFile, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*licOutput, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("License generated:\n")
		fmt.Printf("  Holder:    %s\n", lic.Holder)
		if lic.ExpiresAt > 0 {
			fmt.Printf("  Expires:   %s\n", time.Unix(lic.ExpiresAt, 0).Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("  Expires:   Never (perpetual)")
		}
		if lic.MaxWorkspaces > 0 {
			fmt.Printf("  Max Workspaces: %d\n", lic.MaxWorkspaces)
		} else {
			fmt.Println("  Max Workspaces: Unlimited")
		}
		if lic.MaxWorkflows > 0 {
			fmt.Printf("  Max Workflows: %d\n", lic.MaxWorkflows)
		} else {
			fmt.Println("  Max Workflows: Unlimited")
		}
		if lic.MaxMCP > 0 {
			fmt.Printf("  Max MCP Servers: %d\n", lic.MaxMCP)
		} else {
			fmt.Println("  Max MCP Servers: Unlimited")
		}
		if lic.MaxSkills > 0 {
			fmt.Printf("  Max Skills: %d\n", lic.MaxSkills)
		} else {
			fmt.Println("  Max Skills: Unlimited")
		}
		fmt.Printf("  Output:    %s\n", *licOutput)

	case "verify":
		verifyCmd.Parse(os.Args[2:])
		lic, err := license.VerifyFromFile(*verifyLicPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Verification FAILED: %v\n", err)
			os.Exit(1)
		}

		if lic.MachineID != "" {
			uuid, err := os.ReadFile("/sys/class/dmi/id/product_uuid")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			if string(uuid) != lic.MachineID {
				fmt.Fprintf(os.Stderr, "Error: Machine ID does not match\n")
				os.Exit(1)
			}
		}

		fmt.Println("License is VALID")
		fmt.Printf("  Holder:    %s\n", lic.Holder)
		if lic.ExpiresAt > 0 {
			fmt.Printf("  Expires:   %s\n", time.Unix(lic.ExpiresAt, 0).Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("  Expires:   Never (perpetual)")
		}
		if lic.MaxWorkspaces > 0 {
			fmt.Printf("  Max Workspaces: %d\n", lic.MaxWorkspaces)
		} else {
			fmt.Println("  Max Workspaces: Unlimited")
		}
		if lic.MaxWorkflows > 0 {
			fmt.Printf("  Max Workflows: %d\n", lic.MaxWorkflows)
		} else {
			fmt.Println("  Max Workflows: Unlimited")
		}
		if lic.MaxMCP > 0 {
			fmt.Printf("  Max MCP Servers: %d\n", lic.MaxMCP)
		} else {
			fmt.Println("  Max MCP Servers: Unlimited")
		}
		if lic.MaxSkills > 0 {
			fmt.Printf("  Max Skills: %d\n", lic.MaxSkills)
		} else {
			fmt.Println("  Max Skills: Unlimited")
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// parseDuration parses a duration string like "30d", "365d", or "0"
func parseDuration(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil
	}
	var days int
	if _, err := fmt.Sscanf(s, "%dd", &days); err != nil || days <= 0 {
		return 0, fmt.Errorf("invalid duration format %q, expected e.g. 30d, 365d, or 0", s)
	}
	return time.Duration(days) * 24 * time.Hour, nil
}
