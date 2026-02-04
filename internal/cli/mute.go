package cli

import (
	"fmt"
	"os"

	"github.com/jcornudella/hotbrew/internal/store"
)

// Mute handles `hotbrew mute <domain>`.
func Mute(st *store.Store, domain string) {
	if domain == "" {
		fmt.Println("Usage: hotbrew mute <domain>")
		fmt.Println("\nExamples:")
		fmt.Println("  hotbrew mute example.com")
		fmt.Println("  hotbrew mute medium.com")
		os.Exit(1)
	}

	if err := st.AddRule("mute_domain", domain, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Error muting domain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔇 Muted: %s\n", domain)
	fmt.Println("  Items from this domain will be excluded from future digests.")
}

// Boost handles `hotbrew boost <tag>`.
func Boost(st *store.Store, tag string) {
	if tag == "" {
		fmt.Println("Usage: hotbrew boost <tag>")
		fmt.Println("\nExamples:")
		fmt.Println("  hotbrew boost ai")
		fmt.Println("  hotbrew boost golang")
		os.Exit(1)
	}

	if err := st.AddRule("boost_tag", tag, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Error boosting tag: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔊 Boosted: %s\n", tag)
	fmt.Println("  Items with this tag will rank higher in future digests.")
}

// Rules handles `hotbrew rules` — lists all active rules.
func Rules(st *store.Store) {
	rules, err := st.ListRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing rules: %v\n", err)
		os.Exit(1)
	}

	if len(rules) == 0 {
		fmt.Println("No rules configured.")
		fmt.Println("\nUse 'hotbrew mute <domain>' or 'hotbrew boost <tag>' to add rules.")
		return
	}

	fmt.Println("☕ Active rules:")
	fmt.Println()
	for _, r := range rules {
		icon := "📋"
		switch r.Kind {
		case "mute_domain", "mute_source":
			icon = "🔇"
		case "boost_tag", "boost_domain":
			icon = "🔊"
		}
		fmt.Printf("  %s #%d %s: %s\n", icon, r.ID, r.Kind, r.Pattern)
	}

	fmt.Print("\nDelete a rule: hotbrew rules --delete <id>\n")
}

// DeleteRule handles `hotbrew rules --delete <id>`.
func DeleteRule(st *store.Store, idStr string) {
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid rule ID: %s\n", idStr)
		os.Exit(1)
	}

	if err := st.DeleteRule(id); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting rule: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Deleted rule #%d\n", id)
}
