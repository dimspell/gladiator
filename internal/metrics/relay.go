package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	PacketIn = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_relay_packets_received_total",
			Help: "Total number of packets received by the relay",
		})

	PacketOut = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_relay_packets_sent_total",
			Help: "Total number of packets sent by the relay",
		})

	ActiveRooms = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gladiator_relay_active_rooms",
			Help: "Current number of active rooms",
		})

	ConnectedPeers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gladiator_relay_connected_peers",
			Help: "Current number of connected peers",
		})

	PeersInRoom = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gladiator_relay_peers_in_room",
			Help: "Current number of peers in the the room",
		},
		[]string{"room_id"},
	)

	BytesSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_relay_bytes_sent_total",
			Help: "Total bytes sent by the relay",
		},
	)

	BytesReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_relay_bytes_received_total",
			Help: "Total bytes received by the relay",
		},
	)

	PacketsDropped = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_relay_packets_dropped_total",
			Help: "Total number of packets dropped by the relay",
		},
	)

	PacketLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gladiator_relay_packet_latency_seconds",
			Help:    "Time taken to relay a packet in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0005, 2, 12),
		},
	)

	RelayErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_relay_errors_total",
			Help: "Total number of relay errors by type",
		},
		[]string{"type"},
	)

	PeerDisconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gladiator_relay_peer_disconnects_total",
			Help: "Total number of peer disconnects by reason",
		},
		[]string{"reason"},
	)

	RelayRoomLifetime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gladiator_relay_room_lifetime_seconds",
			Help:    "Lifetime of relay rooms in seconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 8),
		},
	)
)

func InitRelay() {
	prometheus.MustRegister(
		PacketIn,
		PacketOut,
		ActiveRooms,
		ConnectedPeers,
		PeersInRoom,
		BytesSent,
		BytesReceived,
		PacketsDropped,
		PacketLatency,
		RelayErrors,
		PeerDisconnects,
		RelayRoomLifetime,
	)
}
