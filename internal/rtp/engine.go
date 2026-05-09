package rtp

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"time"
)

type Stats struct {
	SentPackets int64
	FailedSend  int64
	JitterMS    float64
	LossPct     float64
}

type Engine struct {
	PacketMS    int
	PayloadSize int
	SSRC        uint32
}

func New(packetMS, payloadSize int) *Engine {
	if packetMS <= 0 {
		packetMS = 20
	}
	if payloadSize <= 0 {
		payloadSize = 160
	}
	return &Engine{
		PacketMS:    packetMS,
		PayloadSize: payloadSize,
		SSRC:        rand.Uint32(),
	}
}

func (e *Engine) Stream(ctx context.Context, remoteAddr string, duration time.Duration) (Stats, error) {
	raddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return Stats{}, fmt.Errorf("resolve rtp addr: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return Stats{}, fmt.Errorf("dial rtp: %w", err)
	}
	defer conn.Close()

	start := time.Now()
	deadline := start.Add(duration)
	interval := time.Duration(e.PacketMS) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var seq uint16
	var ts uint32
	var jitterSum float64
	var jitterN int64
	nextExpected := start.Add(interval)
	stats := Stats{}

	for {
		select {
		case <-ctx.Done():
			stats.LossPct = calculateLoss(stats.SentPackets, stats.FailedSend)
			stats.JitterMS = meanJitter(jitterSum, jitterN)
			return stats, nil
		case now := <-ticker.C:
			if now.After(deadline) {
				stats.LossPct = calculateLoss(stats.SentPackets, stats.FailedSend)
				stats.JitterMS = meanJitter(jitterSum, jitterN)
				return stats, nil
			}

			deviation := now.Sub(nextExpected).Abs().Seconds() * 1000
			jitterSum += deviation
			jitterN++
			nextExpected = nextExpected.Add(interval)

			packet := make([]byte, 12+e.PayloadSize)
			packet[0] = 0x80
			packet[1] = 0x00
			binary.BigEndian.PutUint16(packet[2:4], seq)
			binary.BigEndian.PutUint32(packet[4:8], ts)
			binary.BigEndian.PutUint32(packet[8:12], e.SSRC)

			if _, err := conn.Write(packet); err != nil {
				stats.FailedSend++
			}
			stats.SentPackets++
			seq++
			ts += 160
		}
	}
}

func calculateLoss(sent, failed int64) float64 {
	if sent == 0 {
		return 0
	}
	return (float64(failed) / float64(sent)) * 100
}

func meanJitter(sum float64, n int64) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
