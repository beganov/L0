package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/beganov/L0/internal/cache"
	"github.com/beganov/L0/internal/config"
	"github.com/beganov/L0/internal/logger"
	"github.com/beganov/L0/internal/metrics"
	"github.com/beganov/L0/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pressly/goose"
)

func GetOrderFromDB(ctx context.Context, pool *pgxpool.Pool, orderID string, selectTimeOut time.Duration) (models.Order, error) {
	dbCtx, cancel := context.WithTimeout(ctx, selectTimeOut)
	defer cancel()
	var o models.Order

	// orders
	err := pool.QueryRow(dbCtx, `SELECT order_uid, track_number, entry, locale, customer_id, internal_signature,
        delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid=$1`, orderID).Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale,
		&o.CustomerID, &o.InternalSignature, &o.DeliveryService, &o.Shardkey,
		&o.SmID, &o.DateCreated, &o.OofShard)
	if err != nil {
		logger.Error(err, "failed to select order from DB")
		metrics.DBErrorsTotal.Inc()
		return models.Order{}, err
	}

	// delivery
	err = pool.QueryRow(dbCtx, `SELECT name, phone, zip, city, address, region, email
            FROM deliveries WHERE order_uid=$1`, orderID).Scan(
		&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip,
		&o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email)
	if err != nil {
		logger.Error(err, "failed to select delivery from DB")
		metrics.DBErrorsTotal.Inc()
		return models.Order{}, err
	}

	// payment
	var paymentDT time.Time
	err = pool.QueryRow(dbCtx, `SELECT transaction, request_id, currency, provider, amount, payment_dt, bank,
            delivery_cost, goods_total, custom_fee FROM payments WHERE order_uid=$1`, orderID).Scan(
		&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency,
		&o.Payment.Provider, &o.Payment.Amount, &paymentDT,
		&o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee)
	if err != nil {
		logger.Error(err, "failed to select payment from DB")
		metrics.DBErrorsTotal.Inc()
		return models.Order{}, err
	}
	o.Payment.PaymentDT = paymentDT.Unix()

	// items
	rows, err := pool.Query(dbCtx, `SELECT chrt_id, track_number, price, rid, name, sale, size,
            total_price, nm_id, brand, status FROM items WHERE order_uid=$1`, orderID)
	if err != nil {
		logger.Error(err, "failed to select items from DB")
		metrics.DBErrorsTotal.Inc()
		return models.Order{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var it models.Items
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid,
			&it.Name, &it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			logger.Error(err, "failed to scan item row")
			metrics.DBErrorsTotal.Inc()
			return models.Order{}, err
		}
		o.Items = append(o.Items, it)
	}

	return o, nil
}

func InitDB(ctx context.Context, dsn string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		metrics.DBErrorsTotal.Inc()
		logger.Fatal(err, "unable to create DB pool")
	}
	if err := pool.Ping(ctx); err != nil {
		metrics.DBErrorsTotal.Inc()
		logger.Fatal(err, "unable to connect to DB")
	}
	return pool
}

// run goose migrations
func RunMigrations(dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		metrics.DBErrorsTotal.Inc()
		logger.Fatal(err, "failed to open db for migrations")
	}
	defer db.Close()

	if err := goose.Up(db, config.MigrationPath); err != nil {
		metrics.DBErrorsTotal.Inc()
		logger.Fatal(err, "failed to run migrations")
	}
}

func SaveOrder(ctx context.Context, pool *pgxpool.Pool, order models.Order) error {
	dbCtx, cancel := context.WithTimeout(ctx, config.InsertTimeOut)
	defer cancel()
	tx, err := pool.Begin(dbCtx)
	if err != nil {
		logger.Error(err, "failed to begin transaction")
		metrics.DBErrorsTotal.Inc()
		return err
	}
	defer tx.Rollback(context.Background())

	// orders
	_, err = tx.Exec(dbCtx,
		`INSERT INTO orders(order_uid, track_number, entry, locale, customer_id, internal_signature, delivery_service, shardkey, sm_id, date_created, oof_shard)
         VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT DO NOTHING`, // already exists, skip
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.CustomerID,
		order.InternalSignature, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		logger.Error(err, "failed to insert order into DB")
		metrics.DBErrorsTotal.Inc()
		return err
	}

	// deliveries
	_, err = tx.Exec(dbCtx,
		`INSERT INTO deliveries(order_uid, name, phone, zip, city, address, region, email)
         VALUES($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT DO NOTHING`,
		order.OrderUID,
		order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		logger.Error(err, "failed to insert delivery into DB")
		metrics.DBErrorsTotal.Inc()
		return err
	}

	// payments
	_, err = tx.Exec(dbCtx,
		`INSERT INTO payments(order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
         VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT DO NOTHING`,
		order.OrderUID,
		order.Payment.Transaction, order.Payment.RequestID,
		order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, time.Unix(order.Payment.PaymentDT, 0).UTC(), order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		logger.Error(err, "failed to insert payment into DB")
		metrics.DBErrorsTotal.Inc()
		return err
	}

	// items
	for _, item := range order.Items {
		_, err := tx.Exec(dbCtx,
			`INSERT INTO items(order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
    		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			 ON CONFLICT DO NOTHING`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price,
			item.Rid, item.Name, item.Sale, item.Size,
			item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			logger.Error(err, "failed to insert items into DB")
			metrics.DBErrorsTotal.Inc()
			return err
		}
	}

	return tx.Commit(dbCtx)
}

func LoadCacheFromDB(ctx context.Context, pool *pgxpool.Pool, cache *cache.OrderCache) error {
	dbCtx, cancel := context.WithTimeout(ctx, config.SelectTimeOut)
	defer cancel()
	rows, err := pool.Query(dbCtx, `SELECT order_uid FROM orders`)
	if err != nil {
		logger.Error(err, "failed to select order_uid from DB")
		metrics.DBErrorsTotal.Inc()
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderID string
		if err := rows.Scan(&orderID); err != nil {
			logger.Error(err, "failed to scan order_uid row")
			metrics.DBErrorsTotal.Inc()
			return err
		}

		o, err := GetOrderFromDB(dbCtx, pool, orderID, config.SelectTimeOut)
		if err != nil {
			metrics.DBErrorsTotal.Inc()
			logger.Error(err, "error cache load")
			continue
		}
		cache.Set(orderID, o)
	}
	return nil
}
