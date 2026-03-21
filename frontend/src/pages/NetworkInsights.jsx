import { useEffect, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Download,
  RefreshCw,
  Shield,
  Wifi,
  Router,
  Globe,
  Clock3,
} from 'lucide-react'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
} from 'recharts'
import { diagnosticsApi, apiUtils } from '../api'
import { Badge, Button, Card, CardHeader, EmptyState, Spinner } from '../components/common'

const severityVariant = {
  critical: 'error',
  warning: 'warning',
  info: 'cyan',
  good: 'success',
}

const connectivityVariant = {
  healthy: 'success',
  fair: 'warning',
  degraded: 'error',
  offline: 'error',
}

export default function NetworkInsights() {
  const [summary, setSummary] = useState(null)
  const [history, setHistory] = useState([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')

  const loadData = async ({ silent = false } = {}) => {
    try {
      if (silent) {
        setRefreshing(true)
      } else {
        setLoading(true)
      }
      setError('')

      const [summaryResponse, historyResponse] = await Promise.all([
        diagnosticsApi.getSummary(),
        diagnosticsApi.getHistory(90),
      ])

      setSummary(summaryResponse.data)
      setHistory(historyResponse.data?.samples || [])
    } catch (err) {
      setError(apiUtils.getErrorMessage(err))
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const chartData = useMemo(() => {
    return history.map((sample) => ({
      time: new Date(sample.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      latency: Number(sample.latency || 0),
      gatewayLatency: Number(sample.gatewayLatency || 0),
      packetLoss: Number(sample.packetLoss || 0),
    }))
  }, [history])

  if (loading) {
    return (
      <div className="min-h-[50vh] flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-dark-300">
          <Spinner size="lg" className="text-primary-400" />
          <p>Loading diagnostics…</p>
        </div>
      </div>
    )
  }

  if (error && !summary) {
    return (
      <EmptyState
        icon={AlertTriangle}
        title="Diagnostics unavailable"
        description={error}
        action={<Button onClick={() => loadData()}>Retry</Button>}
      />
    )
  }

  const connectivity = summary?.connectivity || 'offline'
  const insights = summary?.insights || []

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white">Network Insights</h1>
          <p className="text-dark-400 mt-1">Pro-style health scoring, recent history, and operator-focused findings.</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Badge variant={connectivityVariant[connectivity] || 'default'} size="lg">
            <Wifi className="w-3.5 h-3.5" />
            {connectivity}
          </Badge>
          <Button variant="secondary" icon={Download} onClick={() => window.open(diagnosticsApi.getHistoryExportUrl('csv', 180), '_blank')}>
            Export CSV
          </Button>
          <Button variant="secondary" icon={RefreshCw} loading={refreshing} onClick={() => loadData({ silent: true })}>
            Refresh
          </Button>
        </div>
      </div>

      {error && (
        <div className="glass-card p-4 border border-yellow-500/30 bg-yellow-500/10 text-yellow-300 text-sm">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        <ScoreCard title="Health score" value={summary?.healthScore} suffix="/100" icon={Activity} tone={summary?.healthScore >= 85 ? 'green' : summary?.healthScore >= 65 ? 'yellow' : 'red'} />
        <ScoreCard title="Security score" value={summary?.securityScore} suffix="/100" icon={Shield} tone={summary?.securityScore >= 85 ? 'green' : summary?.securityScore >= 65 ? 'yellow' : 'red'} />
        <ScoreCard title="WAN latency" value={summary?.latencyMs?.toFixed?.(1) || '0.0'} suffix=" ms" icon={Globe} tone="blue" />
        <ScoreCard title="Gateway latency" value={summary?.gatewayLatencyMs?.toFixed?.(1) || '0.0'} suffix=" ms" icon={Router} tone="purple" />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        <Card className="xl:col-span-2">
          <CardHeader title="Recent telemetry" icon={Clock3} />
          {chartData.length > 0 ? (
            <div className="h-80">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData}>
                  <CartesianGrid stroke="#1f2937" strokeDasharray="3 3" />
                  <XAxis dataKey="time" stroke="#6b7280" minTickGap={28} />
                  <YAxis stroke="#6b7280" />
                  <Tooltip contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: '12px' }} />
                  <Legend />
                  <Line type="monotone" dataKey="latency" stroke="#f97316" strokeWidth={2} dot={false} name="WAN latency" />
                  <Line type="monotone" dataKey="gatewayLatency" stroke="#22d3ee" strokeWidth={2} dot={false} name="Gateway latency" />
                  <Line type="monotone" dataKey="packetLoss" stroke="#ef4444" strokeWidth={2} dot={false} name="Packet loss %" />
                </LineChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <EmptyState title="No telemetry samples yet" description="Open the dashboard or wait a few seconds for live snapshots to accumulate." />
          )}
        </Card>

        <Card>
          <CardHeader title="Current path" icon={Router} />
          <div className="space-y-4 text-sm">
            <KeyValue label="Primary interface" value={summary?.primaryInterface || '--'} />
            <KeyValue label="Connection type" value={summary?.connectionType || '--'} />
            <KeyValue label="Local IP" value={summary?.localIp || '--'} mono />
            <KeyValue label="Gateway" value={summary?.gateway || '--'} mono />
            <KeyValue label="Public IP" value={summary?.publicIp || '--'} mono />
            <KeyValue label="NAT type" value={summary?.natType || '--'} />
            <KeyValue label="DNS servers" value={summary?.dnsServers?.length ? summary.dnsServers.join(', ') : '--'} mono />
          </div>
        </Card>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        <Card className="xl:col-span-2">
          <CardHeader title="Operator findings" icon={summary?.healthScore >= 80 ? CheckCircle2 : AlertTriangle} />
          <div className="space-y-3">
            {insights.map((insight, index) => (
              <div key={`${insight.title}-${index}`} className="rounded-xl border border-dark-800/70 bg-dark-900/40 p-4">
                <div className="flex items-center justify-between gap-3 mb-2">
                  <h3 className="font-semibold text-white">{insight.title}</h3>
                  <Badge variant={severityVariant[insight.severity] || 'default'}>{insight.severity}</Badge>
                </div>
                <p className="text-sm text-dark-300">{insight.detail}</p>
              </div>
            ))}
          </div>
        </Card>

        <Card>
          <CardHeader title="Inventory" icon={Wifi} />
          <div className="space-y-4 text-sm">
            <KeyValue label="All interfaces" value={summary?.interfaceCounts?.all ?? 0} />
            <KeyValue label="Ethernet" value={summary?.interfaceCounts?.ethernet ?? 0} />
            <KeyValue label="Wi-Fi" value={summary?.interfaceCounts?.wifi ?? 0} />
            <KeyValue label="Virtual / VPN" value={summary?.interfaceCounts?.virtual ?? 0} />
            <div className="pt-4 border-t border-dark-800/70">
              <p className="text-xs uppercase tracking-wide text-dark-500 mb-2">Security anomalies</p>
              {summary?.securityAnomalies?.length ? (
                <ul className="space-y-2 text-dark-300 list-disc list-inside">
                  {summary.securityAnomalies.map((item, index) => (
                    <li key={`${item}-${index}`}>{item}</li>
                  ))}
                </ul>
              ) : (
                <p className="text-dark-400">No active anomalies reported.</p>
              )}
            </div>
          </div>
        </Card>
      </div>
    </motion.div>
  )
}

function ScoreCard({ title, value, suffix = '', icon: Icon, tone = 'blue' }) {
  const toneClasses = {
    green: 'from-green-500/20 to-emerald-500/10 text-green-400',
    yellow: 'from-yellow-500/20 to-orange-500/10 text-yellow-400',
    red: 'from-red-500/20 to-rose-500/10 text-red-400',
    blue: 'from-cyan-500/20 to-blue-500/10 text-cyan-400',
    purple: 'from-purple-500/20 to-fuchsia-500/10 text-purple-400',
  }

  return (
    <div className={`glass-card p-5 bg-gradient-to-br ${toneClasses[tone] || toneClasses.blue}`}>
      <div className="flex items-center justify-between mb-4">
        <p className="text-sm text-dark-300">{title}</p>
        <Icon className="w-5 h-5" />
      </div>
      <p className="text-3xl font-bold text-white">
        {value}
        <span className="text-base text-dark-400">{suffix}</span>
      </p>
    </div>
  )
}

function KeyValue({ label, value, mono = false }) {
  return (
    <div className="flex items-start justify-between gap-4 py-2 border-b border-dark-800/50 last:border-0">
      <span className="text-dark-400">{label}</span>
      <span className={`text-right text-white ${mono ? 'font-mono text-xs break-all' : ''}`}>{value}</span>
    </div>
  )
}
