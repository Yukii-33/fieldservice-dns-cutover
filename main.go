package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	modeFlag := flag.String("mode", string(ModeApply), "change mode: apply or rollback")
	orderPath := flag.String("order", "", "path to a work-order JSON file")
	flag.Parse()

	if *orderPath == "" {
		exitf("-order is required")
	}
	raw, err := os.ReadFile(*orderPath)
	if err != nil {
		exitf("read work order: %v", err)
	}
	var order WorkOrder
	if err := json.Unmarshal(raw, &order); err != nil {
		exitf("decode work order: %v", err)
	}

	change, err := PlanChange(order, ChangeMode(*modeFlag))
	if err != nil {
		exitf("plan DNS change: %v", err)
	}
	client := InfraiClient{APIKey: os.Getenv("INFRAI_API_KEY")}
	ctx := context.Background()
	zoneID, err := client.GetZoneID(ctx, change.Domain)
	if err != nil {
		exitf("resolve DNS zone: %v", err)
	}
	if err := client.UpsertARecord(ctx, zoneID, change); err != nil {
		exitf("write DNS record: %v", err)
	}

	output, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		exitf("encode result: %v", err)
	}
	fmt.Println(string(output))
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
