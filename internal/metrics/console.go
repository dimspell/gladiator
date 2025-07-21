package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	startTime = time.Now()

	Uptime = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "gladiator_uptime_seconds",
			Help: "Console server uptime in seconds",
		}, func() float64 {
			return time.Since(startTime).Seconds()
		})

	ConnectionErrs = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_websocket_connection_errors",
			Help: "Number of connection errors",
		})

	ActiveSessions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gladiator_multiplayer_active_sessions",
			Help: "Current number of active multiplayer sessions (connected players)",
		},
	)

	TotalSessions = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_total_sessions",
			Help: "Total number of multiplayer sessions ever created",
		},
	)

	MultiplayerActiveRooms = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gladiator_multiplayer_active_rooms",
			Help: "Current number of active multiplayer rooms",
		},
	)

	MultiplayerTotalRoomsCreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_total_rooms_created",
			Help: "Total number of multiplayer rooms ever created",
		},
	)

	MessagesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_messages_received_total",
			Help: "Total number of messages received by type",
		},
		[]string{"type"},
	)

	MessagesBroadcasted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_messages_broadcasted_total",
			Help: "Total number of messages broadcasted to all players",
		},
	)

	RoomJoins = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_room_joins_total",
			Help: "Total number of room join events",
		},
	)

	RoomLeaves = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_room_leaves_total",
			Help: "Total number of room leave events",
		},
	)

	MultiplayerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_errors_total",
			Help: "Total number of multiplayer errors by type",
		},
		[]string{"type"},
	)

	PlayersPerRoom = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gladiator_multiplayer_players_per_room",
			Help: "Number of players in each room",
		},
		[]string{"room_id"},
	)

	RoomLifetime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gladiator_multiplayer_room_lifetime_seconds",
			Help:    "Lifetime of rooms in seconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 8),
		},
	)

	PlayerSessionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gladiator_multiplayer_player_session_duration_seconds",
			Help:    "Duration of player sessions in seconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 8),
		},
	)

	MessagesSentPerPlayer = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_messages_sent_per_player_total",
			Help: "Total number of messages sent per player",
		},
		[]string{"user_id"},
	)

	FailedMessageSends = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_failed_message_sends_total",
			Help: "Total number of failed message sends per player and reason",
		},
		[]string{"user_id", "reason"},
	)

	MessageProcessingLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gladiator_multiplayer_message_processing_latency_seconds",
			Help:    "Latency of message processing in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
		},
	)

	RoomReadyEvents = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_room_ready_events_total",
			Help: "Total number of room ready events",
		},
	)

	HostMigrations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_host_migrations_total",
			Help: "Total number of host migrations in rooms",
		},
	)

	WebSocketDisconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_websocket_disconnects_total",
			Help: "Total number of websocket disconnects by reason",
		},
		[]string{"reason"},
	)

	ReconnectAttempts = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_reconnect_attempts_total",
			Help: "Total number of reconnect attempts",
		},
	)

	UnhandledMessageTypes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_unhandled_message_types_total",
			Help: "Total number of unhandled message types",
		},
		[]string{"type"},
	)

	InvalidPayloads = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_multiplayer_invalid_payloads_total",
			Help: "Total number of invalid payloads received",
		},
	)
)

func InitConsole() {
	prometheus.MustRegister(Uptime, ConnectionErrs)
}

func InitMultiplayer() {
	prometheus.MustRegister(
		ActiveSessions,
		TotalSessions,
		MultiplayerActiveRooms,
		MultiplayerTotalRoomsCreated,
		MessagesReceived,
		MessagesBroadcasted,
		RoomJoins,
		RoomLeaves,
		MultiplayerErrors,
		PlayersPerRoom,
		RoomLifetime,
		PlayerSessionDuration,
		MessagesSentPerPlayer,
		FailedMessageSends,
		MessageProcessingLatency,
		RoomReadyEvents,
		HostMigrations,
		WebSocketDisconnects,
		ReconnectAttempts,
		UnhandledMessageTypes,
		InvalidPayloads,
	)
}
