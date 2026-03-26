package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/evoevo/trusted-evidence-engine/internal/engine"
	"github.com/evoevo/trusted-evidence-engine/internal/providers"
	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

type stringListFlag []string

func (flagValues *stringListFlag) String() string {
	return strings.Join(*flagValues, ",")
}

func (flagValues *stringListFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*flagValues = append(*flagValues, item)
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "resolve":
		if err := runResolve(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "tee resolve: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func runResolve(args []string) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	query := fs.String("query", "", "query text")
	claim := fs.String("claim", "", "claim text")
	outputFormat := fs.String("format", "pretty", "output format: pretty or json")
	var sourceURLs stringListFlag
	fs.Var(&sourceURLs, "source-url", "source url (repeatable or comma-separated)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	e := engine.New(providers.DefaultProviders())
	pack, err := e.Resolve(context.Background(), schema.ResolveInput{
		Query:      strings.TrimSpace(*query),
		Claim:      strings.TrimSpace(*claim),
		SourceURLs: sourceURLs,
	})
	if err != nil {
		return err
	}

	if strings.EqualFold(strings.TrimSpace(*outputFormat), "json") {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(pack)
	}

	fmt.Printf("fetched_at: %s\n", pack.FetchedAt.Format("2006-01-02 15:04:05 MST"))
	if pack.Query != "" {
		fmt.Printf("query: %s\n", pack.Query)
	}
	if pack.Claim != "" {
		fmt.Printf("claim: %s\n", pack.Claim)
	}
	if len(pack.SourceURLs) > 0 {
		fmt.Printf("source_urls: %d\n", len(pack.SourceURLs))
	}
	fmt.Printf("items: %d\n\n", len(pack.Items))
	for i, item := range pack.Items {
		fmt.Printf("%d. %s\n", i+1, item.Title)
		fmt.Printf("   url: %s\n", item.URL)
		fmt.Printf("   source_type: %s\n", item.SourceType)
		fmt.Printf("   provider: %s\n", item.Provider)
		fmt.Printf("   reliability: %.2f\n", item.Reliability)
		if item.Snippet != "" {
			fmt.Printf("   snippet: %s\n", item.Snippet)
		}
		fmt.Println()
	}
	return nil
}

func printUsage() {
	fmt.Println("Trusted Evidence Engine CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tee resolve --query \"Did this event already happen?\"")
	fmt.Println("  tee resolve --claim \"This outcome is already settled\" --format json")
	fmt.Println("  tee resolve --source-url \"https://polymarket.com/event/example-market\" --format json")
}
