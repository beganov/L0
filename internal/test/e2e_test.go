package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/beganov/L0/internal/config"
	"github.com/beganov/L0/internal/models"

	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestE2E_OrderFlow_Advanced(t *testing.T) {
	require.NoError(t, godotenv.Load("../../.env"))
	config.VarsInit()

	cases := []struct {
		name       string
		order      any
		wantHTTP   int
		wantCached bool
	}{
		{
			name: "valid order",
			order: models.Order{
				OrderUID:    "test1",
				TrackNumber: "trk1",
				Delivery: models.Delivery{
					Name: "Alice", Phone: "123", Zip: "11111", City: "City", Address: "Street", Region: "Region", Email: "a@a.com",
				},
				Payment: models.Payment{
					Transaction: "txn1", RequestID: "r1", Currency: "USD", Provider: "PP", Amount: 100, PaymentDT: time.Now().Unix(),
				},
				Items: []models.Items{{ChrtID: 1, TrackNumber: "trk1", Price: 100, Name: "Item1", TotalPrice: 100, NmID: 1, Status: 1}},
			},
			wantHTTP:   http.StatusOK,
			wantCached: true,
		},
		{
			name:       "invalid JSON",
			order:      []byte(`{"order_uid": "bad1", "payment":`),
			wantHTTP:   http.StatusNotFound,
			wantCached: false,
		},
		{
			name: "duplicate order UID",
			order: models.Order{
				OrderUID:    "dup1",
				TrackNumber: "trkdup",
				Delivery:    models.Delivery{Name: "Dup"},
				Payment:     models.Payment{Transaction: "txndup"},
				Items:       []models.Items{{ChrtID: 1}},
			},
			wantHTTP:   http.StatusOK,
			wantCached: true,
		},
		{
			name: "bulk items",
			order: func() models.Order {
				items := make([]models.Items, 100)
				for i := 0; i < 100; i++ {
					items[i] = models.Items{
						ChrtID:      i + 1,
						TrackNumber: "trkbulk",
						Price:       10,
						Name:        "Item" + strconv.Itoa(i+1),
						TotalPrice:  10,
						NmID:        i + 1000,
						Status:      1,
					}
				}
				return models.Order{
					OrderUID:    "bulk1",
					TrackNumber: "trkbulk1",
					Delivery:    models.Delivery{Name: "BulkUser"},
					Payment:     models.Payment{Transaction: "txnbulk"},
					Items:       items,
				}
			}(),
			wantHTTP:   http.StatusOK,
			wantCached: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &kafka.Writer{
				Addr:     kafka.TCP(config.KafkaBroker),
				Topic:    config.KafkaTopic,
				Balancer: &kafka.LeastBytes{},
			}
			defer w.Close()

			var val []byte
			var err error
			switch o := tc.order.(type) {
			case models.Order:
				val, err = json.Marshal(o)
				require.NoError(t, err)
			case []byte:
				val = o
			default:
				t.Fatalf("unsupported order type")
			}

			err = w.WriteMessages(context.Background(),
				kafka.Message{Key: []byte("key-" + strconv.FormatInt(time.Now().UnixNano(), 10)), Value: val},
			)
			require.NoError(t, err)

			time.Sleep(2 * time.Second)

			orderUID := ""
			if o, ok := tc.order.(models.Order); ok {
				orderUID = o.OrderUID
			} else if tc.name == "invalid JSON" {
				orderUID = "bad1"
			}

			resp, err := http.Get("http://localhost:8081/order/" + orderUID)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.wantHTTP, resp.StatusCode)

			if tc.wantHTTP == http.StatusOK {
				var got models.Order
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
				require.Equal(t, orderUID, got.OrderUID)
			}
		})
	}
}

func TestE2E_OrderFlow_Stress(t *testing.T) {
	const N = 5
	require.NoError(t, godotenv.Load("../../.env"))
	config.VarsInit()

	w := &kafka.Writer{
		Addr:     kafka.TCP(config.KafkaBroker),
		Topic:    config.KafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

	wg := sync.WaitGroup{}
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			order := generateRandomOrder(i)
			val, err := json.Marshal(order)
			require.NoError(t, err)

			err = w.WriteMessages(context.Background(),
				kafka.Message{Key: []byte(order.OrderUID), Value: val},
			)
			require.NoError(t, err)
			t.Logf("Sent order %s", order.OrderUID)
		}(i)
	}

	wg.Wait()

	time.Sleep(5 * time.Second)
}

func generateRandomOrder(i int) models.Order {
	return models.Order{
		OrderUID:    fmt.Sprintf("stress-%d-%d", i, time.Now().UnixNano()),
		TrackNumber: fmt.Sprintf("trk-%d", i),
		Delivery: models.Delivery{
			Name:    fmt.Sprintf("User%d", i),
			Phone:   "1234567890",
			Zip:     "11111",
			City:    "City",
			Address: "Street 1",
			Region:  "Region",
			Email:   fmt.Sprintf("user%d@example.com", i),
		},
		Payment: models.Payment{
			Transaction:  fmt.Sprintf("txn-%d", i),
			RequestID:    fmt.Sprintf("req-%d", i),
			Currency:     "USD",
			Provider:     "PP",
			Amount:       100 + i,
			PaymentDT:    time.Now().Unix(),
			Bank:         "Bank",
			DeliveryCost: 10,
			GoodsTotal:   90 + i,
		},
		Items: []models.Items{
			{
				ChrtID:      i + 1,
				TrackNumber: fmt.Sprintf("trk-%d", i),
				Price:       90 + i,
				Name:        fmt.Sprintf("Item%d", i),
				TotalPrice:  90 + i,
				NmID:        i + 1000,
				Status:      1,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        fmt.Sprintf("cust%d", i),
		DeliveryService:   "DHL",
		Shardkey:          "shard1",
		SmID:              1,
		DateCreated:       time.Now(),
		OofShard:          "oof1",
	}
}
