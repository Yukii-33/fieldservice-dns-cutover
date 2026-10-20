package main

import (
	"errors"
	"fmt"
)

const cutoverTTL = 60

type WorkOrder struct {
	ID              string   `json:"id"`
	Domain          string   `json:"domain"`
	Hostname        string   `json:"hostname"`
	CurrentAddress  string   `json:"current_address"`
	NewAddress      string   `json:"new_address"`
	DispatchStatus  string   `json:"dispatch_status"`
	Photos          []string `json:"photos"`
	TechnicianNotes string   `json:"technician_follow_up"`
}

type ChangeMode string

const (
	ModeApply    ChangeMode = "apply"
	ModeRollback ChangeMode = "rollback"
)

type DNSChange struct {
	WorkOrderID string `json:"work_order_id"`
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	Content     string `json:"content"`
	Previous    string `json:"previous_content"`
	TTL         int    `json:"ttl"`
	Mode        string `json:"mode"`
}

func PlanChange(order WorkOrder, mode ChangeMode) (DNSChange, error) {
	if order.ID == "" || order.Domain == "" || order.Hostname == "" {
		return DNSChange{}, errors.New("id, domain, and hostname are required")
	}
	if order.CurrentAddress == "" || order.NewAddress == "" {
		return DNSChange{}, errors.New("current_address and new_address are required")
	}

	change := DNSChange{
		WorkOrderID: order.ID,
		Domain:      order.Domain,
		Name:        order.Hostname,
		TTL:         cutoverTTL,
		Mode:        string(mode),
	}

	switch mode {
	case ModeApply:
		if order.DispatchStatus != "completed" {
			return DNSChange{}, fmt.Errorf("dispatch status must be completed, got %q", order.DispatchStatus)
		}
		if len(order.Photos) == 0 {
			return DNSChange{}, errors.New("at least one work-order photo is required")
		}
		if order.TechnicianNotes == "" {
			return DNSChange{}, errors.New("technician follow-up is required")
		}
		change.Content = order.NewAddress
		change.Previous = order.CurrentAddress
	case ModeRollback:
		change.Content = order.CurrentAddress
		change.Previous = order.NewAddress
	default:
		return DNSChange{}, fmt.Errorf("mode must be %q or %q", ModeApply, ModeRollback)
	}

	return change, nil
}
