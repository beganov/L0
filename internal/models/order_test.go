package models_test

import (
	"testing"
	"time"

	"github.com/beganov/L0/internal/models"
)

func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		order   models.Order
		wantErr bool
	}{
		{
			name: "valid order",
			order: models.Order{
				OrderUID:    "123",
				TrackNumber: "TRACK123",
				Entry:       "web",
				Delivery:    models.Delivery{Name: "Ivan"},
				Payment:     models.Payment{Transaction: "TX1"},
				Items: []models.Items{
					{ChrtID: 1, Name: "item1"},
				},
				Locale:          "ru",
				CustomerID:      "cust1",
				DeliveryService: "dhl",
				Shardkey:        "1",
				SmID:            1,
				DateCreated:     time.Now(),
				OofShard:        "1",
			},
			wantErr: false,
		},
		{
			name: "missing order_uid",
			order: models.Order{
				Payment:  models.Payment{Transaction: "TX1"},
				Delivery: models.Delivery{Name: "Ivan"},
				Items:    []models.Items{{ChrtID: 1}},
			},
			wantErr: true,
		},
		{
			name: "missing payment.transaction",
			order: models.Order{
				OrderUID: "123",
				Delivery: models.Delivery{Name: "Ivan"},
				Items:    []models.Items{{ChrtID: 1}},
			},
			wantErr: true,
		},
		{
			name: "missing delivery.name",
			order: models.Order{
				OrderUID: "123",
				Payment:  models.Payment{Transaction: "TX1"},
				Items:    []models.Items{{ChrtID: 1}},
			},
			wantErr: true,
		},
		{
			name: "no items",
			order: models.Order{
				OrderUID: "123",
				Payment:  models.Payment{Transaction: "TX1"},
				Delivery: models.Delivery{Name: "Ivan"},
			},
			wantErr: true,
		},
		{
			name: "item missing chrt_id",
			order: models.Order{
				OrderUID: "123",
				Payment:  models.Payment{Transaction: "TX1"},
				Delivery: models.Delivery{Name: "Ivan"},
				Items:    []models.Items{{ChrtID: 0, Name: "bad"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
