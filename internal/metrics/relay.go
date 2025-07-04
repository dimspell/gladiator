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
)

func InitRelay() {
	prometheus.MustRegister(PacketIn, PacketOut, ActiveRooms, ConnectedPeers, PeersInRoom)
}
