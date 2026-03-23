package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"inventory-desktop/internal/backend"
)

func main() {
	dbPath := flag.String("db", "", "SQLite DB path (optional; defaults to app config path)")
	mode := flag.String("mode", "standalone", "Deployment mode: standalone|lan_sync")
	flag.Parse()

	cfg, err := backend.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve default config: %v\n", err)
		os.Exit(1)
	}
	if *dbPath != "" {
		cfg.DBPath = *dbPath
	} else if envDBPath := strings.TrimSpace(os.Getenv("MYDUKA_DB_PATH")); envDBPath != "" {
		cfg.DBPath = envDBPath
	}
	cfg.Mode = backend.DeploymentMode(*mode)

	svc, err := backend.NewService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create backend service: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = svc.Close() }()

	if err := svc.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "start backend service: %v\n", err)
		os.Exit(1)
	}

	result, err := svc.SeedDemoData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed demo data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seed completed for %q\n", result.BusinessName)
	fmt.Printf("Staff added: %d\n", result.StaffAdded)
	fmt.Printf("Categories added: %d\n", result.CategoriesAdded)
	fmt.Printf("Suppliers added: %d\n", result.SuppliersAdded)
	fmt.Printf("Products added: %d\n", result.ProductsAdded)
	fmt.Printf("Purchase orders added: %d\n", result.OrdersAdded)
	fmt.Printf("Sales added: %d\n", result.SalesAdded)
	fmt.Println("Demo credentials:")
	for _, c := range result.Credentials {
		fmt.Printf("- %s (%s) username: %s password: %s [%s]\n", c.Name, c.Role, c.Username, c.Password, c.Notes)
	}
}
