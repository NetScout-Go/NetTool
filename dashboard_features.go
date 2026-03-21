package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NetScout-Go/NetTool/app/core"
)

type networkHistoryStore struct {
	mu      sync.RWMutex
	maxSize int
	samples []core.NetworkInfo
}

type diagnosticInsight struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

type diagnosticsSummary struct {
	GeneratedAt       time.Time             `json:"generatedAt"`
	HealthScore       int                   `json:"healthScore"`
	SecurityScore     int                   `json:"securityScore"`
	Connectivity      string                `json:"connectivity"`
	PrimaryInterface  string                `json:"primaryInterface"`
	ConnectionType    string                `json:"connectionType"`
	PublicIP          string                `json:"publicIp,omitempty"`
	LocalIP           string                `json:"localIp,omitempty"`
	Gateway           string                `json:"gateway,omitempty"`
	NatType           string                `json:"natType,omitempty"`
	LatencyMs         float64               `json:"latencyMs"`
	GatewayLatencyMs  float64               `json:"gatewayLatencyMs"`
	PacketLossPercent float64               `json:"packetLossPercent"`
	DnsServers        []string              `json:"dnsServers"`
	InterfaceCounts   map[string]int        `json:"interfaceCounts"`
	Insights          []diagnosticInsight   `json:"insights"`
	SecurityAnomalies []string              `json:"securityAnomalies"`
	Integrity         *core.IntegrityStatus `json:"integrity,omitempty"`
	RecentSamples     []core.NetworkInfo    `json:"recentSamples,omitempty"`
}

var historyStore = &networkHistoryStore{maxSize: 240}

func recordNetworkSnapshot(info *core.NetworkInfo) {
	if info == nil {
		return
	}

	copyInfo := *info

	historyStore.mu.Lock()
	defer historyStore.mu.Unlock()

	historyStore.samples = append(historyStore.samples, copyInfo)
	if len(historyStore.samples) > historyStore.maxSize {
		historyStore.samples = append([]core.NetworkInfo(nil), historyStore.samples[len(historyStore.samples)-historyStore.maxSize:]...)
	}
}

func getRecentNetworkSnapshots(limit int) []core.NetworkInfo {
	historyStore.mu.RLock()
	defer historyStore.mu.RUnlock()

	if limit <= 0 || limit > len(historyStore.samples) {
		limit = len(historyStore.samples)
	}
	if limit == 0 {
		return []core.NetworkInfo{}
	}

	start := len(historyStore.samples) - limit
	result := append([]core.NetworkInfo(nil), historyStore.samples[start:]...)
	return result
}

func exportNetworkSnapshotsCSV(limit int) (string, error) {
	samples := getRecentNetworkSnapshots(limit)
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	headers := []string{
		"timestamp", "interface_name", "connection_type", "connected", "ipv4", "gateway", "public_ip", "nat_type",
		"latency_ms", "gateway_latency_ms", "packet_loss_percent", "bytes_sent", "bytes_received", "packets_sent", "packets_received",
	}

	if err := writer.Write(headers); err != nil {
		return "", err
	}

	for _, sample := range samples {
		record := []string{
			sample.Timestamp.Format(time.RFC3339),
			sample.InterfaceName,
			sample.ConnectionType,
			fmt.Sprintf("%t", sample.Connected),
			sample.Ipv4,
			sample.Gateway,
			sample.PublicIp,
			sample.NatType,
			fmt.Sprintf("%.2f", sample.Latency),
			fmt.Sprintf("%.2f", sample.GatewayLatency),
			fmt.Sprintf("%.2f", sample.PacketLoss),
			fmt.Sprintf("%d", sample.BytesSent),
			fmt.Sprintf("%d", sample.BytesReceived),
			fmt.Sprintf("%d", sample.PacketsSent),
			fmt.Sprintf("%d", sample.PacketsReceived),
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func buildDiagnosticsSummary() (*diagnosticsSummary, error) {
	info, err := core.GetNetworkInfo()
	if err != nil {
		return nil, err
	}
	info.Timestamp = time.Now()
	recordNetworkSnapshot(info)

	interfaces, _ := core.GetInterfaces()
	security := core.PerformSecurityChecks()
	integrity := core.GetCachedIntegrityStatus()
	if integrity == nil {
		integrity = core.VerifyBinaryIntegrity()
	}

	summary := &diagnosticsSummary{
		GeneratedAt:       time.Now(),
		Connectivity:      connectivityLabel(info),
		PrimaryInterface:  info.InterfaceName,
		ConnectionType:    info.ConnectionType,
		PublicIP:          info.PublicIp,
		LocalIP:           info.Ipv4,
		Gateway:           info.Gateway,
		NatType:           info.NatType,
		LatencyMs:         info.Latency,
		GatewayLatencyMs:  info.GatewayLatency,
		PacketLossPercent: info.PacketLoss,
		DnsServers:        append([]string(nil), info.DnsServers...),
		Insights:          buildInsights(info, security),
		SecurityAnomalies: append([]string(nil), security.Anomalies...),
		Integrity:         integrity,
		RecentSamples:     getRecentNetworkSnapshots(30),
		InterfaceCounts:   map[string]int{"all": 0, "ethernet": 0, "wifi": 0, "virtual": 0},
	}

	summary.HealthScore = calculateHealthScore(info)
	summary.SecurityScore = calculateSecurityScore(security, integrity)

	if interfaces != nil {
		summary.InterfaceCounts["all"] = len(interfaces.All)
		summary.InterfaceCounts["ethernet"] = len(interfaces.Ethernet)
		summary.InterfaceCounts["wifi"] = len(interfaces.WiFi)
		summary.InterfaceCounts["virtual"] = len(interfaces.Virtual)
	}

	return summary, nil
}

func connectivityLabel(info *core.NetworkInfo) string {
	if info == nil || !info.Connected {
		return "offline"
	}
	if info.PacketLoss >= 5 || info.Latency >= 150 {
		return "degraded"
	}
	if info.PacketLoss > 0 || info.Latency >= 60 {
		return "fair"
	}
	return "healthy"
}

func calculateHealthScore(info *core.NetworkInfo) int {
	if info == nil {
		return 0
	}

	score := 100
	if !info.Connected {
		score -= 60
	}
	if info.Latency >= 200 {
		score -= 25
	} else if info.Latency >= 100 {
		score -= 15
	} else if info.Latency >= 50 {
		score -= 8
	}

	if info.PacketLoss >= 5 {
		score -= 25
	} else if info.PacketLoss >= 1 {
		score -= 12
	} else if info.PacketLoss > 0 {
		score -= 5
	}

	if info.Gateway == "" || info.Gateway == "N/A" {
		score -= 8
	}
	if info.PublicIp == "" {
		score -= 5
	}
	if info.DoubleNat {
		score -= 8
	}
	if info.BehindCgnat {
		score -= 10
	}

	if score < 0 {
		return 0
	}
	return score
}

func calculateSecurityScore(security *core.SecurityCheckResult, integrity *core.IntegrityStatus) int {
	score := 100
	if security != nil {
		anomalyPenalty := len(security.Anomalies) * 12
		score -= anomalyPenalty
		if !security.Passed {
			score -= 10
		}
	}
	if integrity != nil {
		if !integrity.Verified {
			score -= 20
		}
		if integrity.TamperDetected || integrity.ShouldBlock {
			score -= 35
		}
	}
	if score < 0 {
		return 0
	}
	return score
}

func buildInsights(info *core.NetworkInfo, security *core.SecurityCheckResult) []diagnosticInsight {
	insights := []diagnosticInsight{}
	if info == nil {
		return insights
	}

	appendInsight := func(severity, title, detail string) {
		insights = append(insights, diagnosticInsight{Severity: severity, Title: title, Detail: detail})
	}

	if !info.Connected {
		appendInsight("critical", "Interface offline", "No active IP connectivity was detected on the primary interface.")
	}
	if info.PacketLoss >= 5 {
		appendInsight("critical", "High packet loss", fmt.Sprintf("Packet loss is %.1f%%. Expect retransmits, poor VoIP quality, and unstable sessions.", info.PacketLoss))
	} else if info.PacketLoss > 0 {
		appendInsight("warning", "Packet loss detected", fmt.Sprintf("Packet loss is %.1f%%. This is small but worth watching if users report instability.", info.PacketLoss))
	}
	if info.Latency >= 120 {
		appendInsight("warning", "High internet latency", fmt.Sprintf("End-to-end latency is %.1f ms, which is high for interactive traffic.", info.Latency))
	}
	if info.GatewayLatency >= 10 {
		appendInsight("warning", "Slow gateway response", fmt.Sprintf("Gateway latency is %.1f ms. Local network congestion or Wi-Fi issues may be involved.", info.GatewayLatency))
	}
	if info.DoubleNat {
		appendInsight("warning", "Double NAT detected", "Multiple NAT layers can break inbound access, VPNs, and peer-to-peer apps.")
	}
	if info.BehindCgnat {
		appendInsight("warning", "Carrier-grade NAT detected", "Public inbound connectivity may be impossible without ISP changes or a tunnel/VPN strategy.")
	}
	if len(info.DnsServers) == 0 {
		appendInsight("warning", "No DNS servers found", "Resolver configuration could not be detected from the host.")
	}
	if info.SwitchDetected {
		appendInsight("info", "Layer-2 path detected", fmt.Sprintf("Switch visibility is available%s.", func() string {
			if info.SwitchPort != "" {
				return " on port " + info.SwitchPort
			}
			return ""
		}()))
	}
	if security != nil && len(security.Anomalies) > 0 {
		anomalies := append([]string(nil), security.Anomalies...)
		sort.Strings(anomalies)
		appendInsight("warning", "Security anomalies present", strings.Join(anomalies, "; "))
	}
	if len(insights) == 0 {
		appendInsight("good", "Network looks healthy", "No obvious connectivity or security anomalies were detected in the latest sample.")
	}

	return insights
}
