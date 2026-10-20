package main

import "testing"

func TestPlanChange(t *testing.T) {
	ready := WorkOrder{
		ID: "wo-1842", Domain: "example.com", Hostname: "dispatch.example.com",
		CurrentAddress: "192.0.2.10", NewAddress: "198.51.100.24",
		DispatchStatus: "completed", Photos: []string{"cabinet-front.jpg"},
		TechnicianNotes: "health check passed from the service van",
	}
	tests := []struct {
		name        string
		order       WorkOrder
		mode        ChangeMode
		wantContent string
		wantErr     bool
	}{
		{name: "ready dispatch cuts over", order: ready, mode: ModeApply, wantContent: "198.51.100.24"},
		{name: "rollback restores captured address", order: ready, mode: ModeRollback, wantContent: "192.0.2.10"},
		{name: "active dispatch cannot cut over", order: withStatus(ready, "en_route"), mode: ModeApply, wantErr: true},
		{name: "photo evidence is required", order: withoutPhotos(ready), mode: ModeApply, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanChange(tt.order, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("PlanChange() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Content != tt.wantContent || got.TTL != 60 {
				t.Fatalf("PlanChange() = content %q, ttl %d", got.Content, got.TTL)
			}
		})
	}
}

func withStatus(order WorkOrder, status string) WorkOrder {
	order.DispatchStatus = status
	return order
}

func withoutPhotos(order WorkOrder) WorkOrder {
	order.Photos = nil
	return order
}
