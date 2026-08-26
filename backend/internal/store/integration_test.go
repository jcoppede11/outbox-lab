//go:build integration

// Package store_test contiene los tests de integración del patrón outbox.
// Corren contra un PostgreSQL real levantado con testcontainers, así que
// necesitan un daemon Docker funcionando. Ejecútalos con:
//
//	go test -tags integration ./...
//
// El test principal reproduce el demo de caos del README y
// verifica el invariante que da valor al patrón: todo pago registrado termina
// notificado, y ningún evento se pierde mientras el broker está caído.
package store_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"outbox-lab/internal/broker"
	"outbox-lab/internal/relay"
	"outbox-lab/internal/store"
)

// newPostgres levanta un contenedor PostgreSQL desechable, aplica la
// migración del esquema y devuelve un pool listo para usar. El contenedor y
// el pool se destruyen automáticamente al terminar el test.
func newPostgres(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	testContext := context.Background()

	postgresContainer, err := tcpostgres.Run(testContext, "postgres:17-alpine",
		tcpostgres.WithDatabase("outbox_lab"),
		tcpostgres.WithUsername("outbox"),
		tcpostgres.WithPassword("outbox"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	testcontainers.CleanupContainer(t, postgresContainer)

	dsn, err := postgresContainer.ConnectionString(testContext, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(testContext, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	applyMigration(t, pool)
	return store.New(pool), pool
}

// applyMigration: ejecuta la migración up que crea las tablas payments y
// outbox, reutilizando el SQL exacto que viene en el repo.
func applyMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	path := filepath.Join("..", "..", "migrations", "000001_init_schema.up.sql")
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	if _, err := pool.Exec(context.Background(), string(sqlBytes)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
}

// discardLogger: devuelve un logger que descarta la salida, para mantener los logs del test silenciosos.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// countOutbox: devuelve cuántas filas de outbox tienen el status dado.
func countOutbox(t *testing.T, pool *pgxpool.Pool, status string) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox WHERE status = $1`, status).Scan(&count)
	if err != nil {
		t.Fatalf("count outbox (%s): %v", status, err)
	}
	return count
}

// TestOutboxSurvivesBrokerOutage es el test principal: reproduce la secuencia
// exacta del README (broker caído → crear pagos → broker arriba → drenar) y
// verifica que nada se pierde y que todo termina entregado.
//
// El relay se ejercita vía store.DrainPending, que es lo que la goroutine del
// relay invoca en cada tick — el ticker no aporta nada al invariante, así que
// testear el drenado directamente mantiene las aserciones deterministas.
func TestOutboxSurvivesBrokerOutage(t *testing.T) {
	testContext := context.Background()
	dataStore, pool := newPostgres(t)
	fakeBroker := broker.New()

	const payments = 5

	// Fase 1 — caos: el broker está caído. Los pagos deben registrarse igual y
	// sus eventos acumularse como pending; nada llega al broker.
	fakeBroker.SetDown(true)
	for i := 0; i < payments; i++ {
		if _, err := dataStore.CreatePayment(testContext, "alice", 99.90, "USD"); err != nil {
			t.Fatalf("create payment %d: %v", i, err)
		}
	}

	// Un drenado con el broker caído no debe publicar nada ni perder nada.
	if count, err := dataStore.DrainPending(testContext, fakeBroker.Publish, 50); err != nil || count != 0 {
		t.Fatalf("drain while down: count=%d err=%v, want count=0 err=nil", count, err)
	}
	if actualCount := len(fakeBroker.Received()); actualCount != 0 {
		t.Fatalf("broker received %d events while down, want 0", actualCount)
	}
	if actualCount := countOutbox(t, pool, "pending"); actualCount != payments {
		t.Fatalf("pending outbox rows = %d, want %d", actualCount, payments)
	}

	// Fase 2 — recuperación: levantar el broker y drenar hasta vaciar la outbox.
	fakeBroker.SetDown(false)
	total := 0
	for {
		count, err := dataStore.DrainPending(testContext, fakeBroker.Publish, 50)
		if err != nil {
			t.Fatalf("drain after recovery: %v", err)
		}
		total += count
		if count == 0 {
			break
		}
	}

	// El invariante: todo pago terminó notificado exactamente una vez, sin pendientes.
	if total != payments {
		t.Fatalf("drained %d events, want %d", total, payments)
	}
	if actualCount := len(fakeBroker.Received()); actualCount != payments {
		t.Fatalf("broker received %d events, want %d", actualCount, payments)
	}
	if actualCount := countOutbox(t, pool, "sent"); actualCount != payments {
		t.Fatalf("sent outbox rows = %d, want %d", actualCount, payments)
	}
	if actualCount := countOutbox(t, pool, "pending"); actualCount != 0 {
		t.Fatalf("pending outbox rows = %d after recovery, want 0", actualCount)
	}
}

// TestRelayGoroutineDrains ejercita el relay real en segundo plano (la
// goroutine con time.Ticker) en lugar de llamar DrainPending directo,
// para demostrar que todo el stack converge solo una vez que el broker está arriba.
func TestRelayGoroutineDrains(t *testing.T) {
	dataStore, pool := newPostgres(t)
	fakeBroker := broker.New()

	testContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	relay := relay.New(dataStore, fakeBroker.Publish, 50*time.Millisecond, 50, discardLogger())
	go relay.Run(testContext)

	const payments = 3
	for i := 0; i < payments; i++ {
		if _, err := dataStore.CreatePayment(testContext, "bob", 10, "USD"); err != nil {
			t.Fatalf("create payment %d: %v", i, err)
		}
	}

	// Consistencia eventual: el relay hace tick en segundo plano, así que
	// hacemos poll hasta que drenó todo o se cumple el deadline.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if len(fakeBroker.Received()) == payments && countOutbox(t, pool, "pending") == 0 {
			return // el relay convergió solo
		}
		if time.Now().After(deadline) {
			t.Fatalf("relay did not drain in time: received=%d pending=%d",
				len(fakeBroker.Received()), countOutbox(t, pool, "pending"))
		}
		time.Sleep(50 * time.Millisecond)
	}
}
