import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  Wifi, 
  WifiOff, 
  Globe, 
  Server,
  Network,
  Cable,
  Router,
  RefreshCw,
  MonitorSmartphone,
  Cpu,
  Shield,
  AlertTriangle,
  Layers,
  ExternalLink
} from 'lucide-react'
import StatsCard from '../components/dashboard/StatsCard'
import InterfaceDetails from '../components/dashboard/InterfaceDetails'
import { networkApi } from '../api'

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1
    }
  }
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.5 }
  }
}

export default function Dashboard({ networkData, connected }) {
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [lastUpdated, setLastUpdated] = useState(null)
  const [localData, setLocalData] = useState(null)

  // Use WebSocket data or fetch manually
  useEffect(() => {
    if (networkData) {
      setLocalData(networkData)
      setLastUpdated(new Date())
    }
  }, [networkData])

  // Initial fetch if no WebSocket data
  useEffect(() => {
    if (!networkData) {
      fetchNetworkInfo()
    }
  }, [])

  const fetchNetworkInfo = async () => {
    try {
      const response = await networkApi.getNetworkInfo()
      setLocalData(response.data)
      setLastUpdated(new Date())
    } catch (error) {
      console.error('Failed to fetch network info:', error)
    }
  }

  const formatUptime = (seconds) => {
    if (!seconds) return '--:--:--'
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = Math.floor(seconds % 60)
    
    if (days > 0) {
      return `${days}d ${hours}h ${minutes}m`
    }
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  const data = localData || {}

  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Network Dashboard</h1>
          <p className="text-dark-400 mt-1">Device network information</p>
        </div>
        <div className="flex items-center gap-4">
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`btn-secondary flex items-center gap-2 ${!autoRefresh && 'opacity-50'}`}
          >
            <RefreshCw className={`w-4 h-4 ${autoRefresh && connected && 'animate-spin'}`} />
            {autoRefresh ? 'Live' : 'Paused'}
          </button>
          {lastUpdated && (
            <span className="text-sm text-dark-400">
              {lastUpdated.toLocaleTimeString()}
            </span>
          )}
        </div>
      </div>

      {/* Connection Status Banner */}
      <motion.div variants={itemVariants}>
        <div className={`glass-card p-4 border-l-4 ${connected ? 'border-l-green-500' : 'border-l-red-500'}`}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className={`p-3 rounded-xl ${connected ? 'bg-green-500/20' : 'bg-red-500/20'}`}>
                {connected ? <Wifi className="w-6 h-6 text-green-400" /> : <WifiOff className="w-6 h-6 text-red-400" />}
              </div>
              <div>
                <h2 className="text-lg font-semibold text-white">
                  {connected ? 'Connected' : 'Disconnected'}
                </h2>
                <p className="text-sm text-dark-400">
                  {data.connectionType || 'Unknown'} • Uptime: {formatUptime(data.uptime)}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-6">
              <div className="text-right">
                <p className="text-xs text-dark-500">Latency</p>
                <p className={`text-xl font-bold ${
                  data.latency < 20 ? 'text-green-400' : 
                  data.latency < 50 ? 'text-yellow-400' : 
                  data.latency < 100 ? 'text-orange-400' : 'text-red-400'
                }`}>
                  {data.latency ? `${data.latency.toFixed(1)} ms` : '-- ms'}
                </p>
              </div>
              <div className="text-right">
                <p className="text-xs text-dark-500">Packet Loss</p>
                <p className={`text-xl font-bold ${
                  data.packetLoss === 0 ? 'text-green-400' : 
                  data.packetLoss < 1 ? 'text-yellow-400' : 'text-red-400'
                }`}>
                  {data.packetLoss !== undefined ? `${data.packetLoss.toFixed(1)}%` : '--%'}
                </p>
              </div>
            </div>
          </div>
        </div>
      </motion.div>

      {/* Main Info Cards */}
      <motion.div variants={itemVariants} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* IP Configuration */}
        <StatsCard
          title="IP Configuration"
          gradient="cyan"
          icon={Globe}
        >
          <div className="space-y-3">
            <div className="flex justify-between items-center py-2 border-b border-dark-800/50">
              <span className="text-dark-400">IPv4 Address</span>
              <span className="font-mono text-white">{data.ipv4 || '--'}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-dark-800/50">
              <span className="text-dark-400">Subnet Mask</span>
              <span className="font-mono text-white">{data.subnet || '--'}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-dark-800/50">
              <span className="text-dark-400">Gateway</span>
              <span className="font-mono text-white">{data.gateway || '--'}</span>
            </div>
            <div className="flex justify-between items-center py-2">
              <span className="text-dark-400">DHCP</span>
              <span className={`font-semibold ${data.dhcpEnabled ? 'text-green-400' : 'text-yellow-400'}`}>
                {data.dhcpEnabled ? 'Enabled' : 'Static'}
              </span>
            </div>
          </div>
        </StatsCard>

        {/* Gateway / Router Info */}
        <StatsCard
          title="Gateway / Router"
          gradient="orange"
          icon={Router}
        >
          <div className="space-y-3">
            <div className="flex justify-between items-center py-2 border-b border-dark-800/50">
              <span className="text-dark-400">Gateway IP</span>
              <span className="font-mono text-white">{data.gateway || '--'}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-dark-800/50">
              <span className="text-dark-400">Gateway MAC</span>
              <span className="font-mono text-xs text-white">{data.gatewayMac || '--'}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-dark-800/50">
              <span className="text-dark-400">Gateway Latency</span>
              <span className="font-semibold text-white">
                {data.gatewayLatency ? `${data.gatewayLatency.toFixed(1)} ms` : '-- ms'}
              </span>
            </div>
            <div className="flex justify-between items-center py-2">
              <span className="text-dark-400">Hops to Internet</span>
              <span className="font-semibold text-white">{data.hopsToInternet || '--'}</span>
            </div>
          </div>
        </StatsCard>

        {/* DNS Servers */}
        <StatsCard
          title="DNS Servers"
          gradient="purple"
          icon={Server}
        >
          <div className="space-y-3">
            {data.dnsServers && data.dnsServers.length > 0 ? (
              data.dnsServers.slice(0, 4).map((dns, index) => (
                <div key={index} className="flex justify-between items-center py-2 border-b border-dark-800/50 last:border-0">
                  <span className="text-dark-400">DNS {index + 1}</span>
                  <span className="font-mono text-white">{dns}</span>
                </div>
              ))
            ) : (
              <div className="text-center py-6 text-dark-400">
                <Server className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p>No DNS servers configured</p>
              </div>
            )}
          </div>
        </StatsCard>
      </motion.div>

      {/* NAT / Public IP Section */}
      <motion.div variants={itemVariants}>
        <div className={`glass-card p-6 border-l-4 ${
          data.natType === 'None' ? 'border-l-green-500' :
          data.natType === 'Single NAT' ? 'border-l-blue-500' :
          data.natType === 'Double NAT' ? 'border-l-yellow-500' :
          data.natType === 'CGNAT' ? 'border-l-red-500' :
          'border-l-dark-600'
        }`}>
          <div className="flex items-center gap-3 mb-6">
            <div className={`p-2 rounded-lg ${
              data.natType === 'None' ? 'bg-green-500/20' :
              data.natType === 'Single NAT' ? 'bg-blue-500/20' :
              data.natType === 'Double NAT' ? 'bg-yellow-500/20' :
              data.natType === 'CGNAT' ? 'bg-red-500/20' :
              'bg-dark-700/50'
            }`}>
              <Shield className={`w-5 h-5 ${
                data.natType === 'None' ? 'text-green-400' :
                data.natType === 'Single NAT' ? 'text-blue-400' :
                data.natType === 'Double NAT' ? 'text-yellow-400' :
                data.natType === 'CGNAT' ? 'text-red-400' :
                'text-dark-400'
              }`} />
            </div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-white">NAT & Public IP</h3>
              <p className="text-sm text-dark-400">Network Address Translation status</p>
            </div>
            {/* NAT Status Badge */}
            <div className={`px-3 py-1.5 rounded-lg text-sm font-medium ${
              data.natType === 'None' ? 'bg-green-500/20 text-green-400' :
              data.natType === 'Single NAT' ? 'bg-blue-500/20 text-blue-400' :
              data.natType === 'Double NAT' ? 'bg-yellow-500/20 text-yellow-400' :
              data.natType === 'CGNAT' ? 'bg-red-500/20 text-red-400' :
              'bg-dark-700/50 text-dark-400'
            }`}>
              {data.natType || 'Detecting...'}
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {/* Public IP */}
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide flex items-center gap-1">
                <ExternalLink className="w-3 h-3" />
                Public IP
              </p>
              <p className="text-lg font-mono text-white">{data.publicIp || 'Fetching...'}</p>
            </div>
            
            {/* NAT Layers */}
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide flex items-center gap-1">
                <Layers className="w-3 h-3" />
                NAT Layers
              </p>
              <p className="text-lg font-semibold text-white">
                {data.natLayers !== undefined ? data.natLayers : '--'}
                {data.natLayers > 1 && <span className="text-yellow-400 text-sm ml-2">(Multiple)</span>}
              </p>
            </div>
            
            {/* First NAT Gateway */}
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide">NAT Gateway</p>
              <p className="text-lg font-mono text-white">{data.natGatewayIp || data.gateway || '--'}</p>
            </div>
            
            {/* External Router (if double NAT) */}
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide">External Router</p>
              <p className="text-lg font-mono text-white">{data.externalRouter || '--'}</p>
            </div>
          </div>

          {/* NAT Warnings */}
          {(data.doubleNat || data.behindCgnat) && (
            <div className={`mt-4 p-3 rounded-lg flex items-start gap-3 ${
              data.behindCgnat ? 'bg-red-500/10 border border-red-500/20' :
              'bg-yellow-500/10 border border-yellow-500/20'
            }`}>
              <AlertTriangle className={`w-5 h-5 mt-0.5 flex-shrink-0 ${
                data.behindCgnat ? 'text-red-400' : 'text-yellow-400'
              }`} />
              <div>
                <p className={`font-medium ${data.behindCgnat ? 'text-red-400' : 'text-yellow-400'}`}>
                  {data.behindCgnat ? 'Carrier-Grade NAT Detected' : 'Double NAT Detected'}
                </p>
                <p className="text-sm text-dark-400 mt-1">
                  {data.behindCgnat 
                    ? 'Your ISP is using CGNAT. This may prevent port forwarding and affect peer-to-peer connections. Contact your ISP for a public IP address.'
                    : 'Multiple NAT layers detected. This can cause issues with port forwarding, VPNs, and gaming. Consider bridging one of your routers.'}
                </p>
              </div>
            </div>
          )}
        </div>
      </motion.div>

      {/* Interface Details */}
      <motion.div variants={itemVariants}>
        <InterfaceDetails data={data} />
      </motion.div>

      {/* Network Interface Card */}
      <motion.div variants={itemVariants}>
        <div className="glass-card gradient-blue p-6">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-2 rounded-lg bg-blue-500/20">
              <Cable className="w-5 h-5 text-blue-400" />
            </div>
            <h3 className="text-lg font-semibold text-white">Network Interface</h3>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide">Interface Name</p>
              <p className="text-lg font-semibold text-white">{data.interfaceName || '--'}</p>
            </div>
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide">MAC Address</p>
              <p className="text-lg font-mono text-white">{data.macAddress || '--'}</p>
            </div>
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide">Link Speed</p>
              <p className="text-lg font-semibold text-white">{data.linkSpeed || '--'}</p>
            </div>
            <div className="space-y-1">
              <p className="text-xs text-dark-500 uppercase tracking-wide">MTU</p>
              <p className="text-lg font-semibold text-white">{data.mtu || '--'}</p>
            </div>
          </div>

          {/* IPv6 Section */}
          {data.ipv6 && (
            <div className="mt-6 pt-6 border-t border-dark-800/50">
              <div className="space-y-1">
                <p className="text-xs text-dark-500 uppercase tracking-wide">IPv6 Address</p>
                <p className="text-sm font-mono text-dark-300 break-all">{data.ipv6}</p>
              </div>
            </div>
          )}
        </div>
      </motion.div>

      {/* Connection Path Visualization */}
      <motion.div variants={itemVariants}>
        <div className="glass-card p-6">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-2 rounded-lg bg-primary-500/20">
              <Network className="w-5 h-5 text-primary-400" />
            </div>
            <h3 className="text-lg font-semibold text-white">Connection Path</h3>
          </div>
          
          <div className="flex items-center justify-center gap-4 py-8 overflow-x-auto">
            {/* This Device */}
            <div className="flex flex-col items-center min-w-[100px]">
              <div className="w-16 h-16 rounded-xl bg-primary-500/20 flex items-center justify-center mb-2">
                <MonitorSmartphone className="w-8 h-8 text-primary-400" />
              </div>
              <p className="text-sm font-medium text-white">This Device</p>
              <p className="text-xs text-dark-400 font-mono">{data.ipv4 || '--'}</p>
            </div>

            {/* Connection Line */}
            <div className="flex items-center">
              <div className="w-8 h-0.5 bg-dark-700"></div>
              <div className={`w-3 h-3 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`}></div>
              <div className="w-8 h-0.5 bg-dark-700"></div>
            </div>

            {/* Switch (if detected) */}
            {data.switchDetected && (
              <>
                <div className="flex flex-col items-center min-w-[100px]">
                  <div className="w-16 h-16 rounded-xl bg-cyan-500/20 flex items-center justify-center mb-2">
                    <Cpu className="w-8 h-8 text-cyan-400" />
                  </div>
                  <p className="text-sm font-medium text-white">Switch</p>
                  <p className="text-xs text-dark-400">Port {data.switchPort || '?'}</p>
                </div>

                <div className="flex items-center">
                  <div className="w-8 h-0.5 bg-dark-700"></div>
                  <div className="w-3 h-3 rounded-full bg-green-500"></div>
                  <div className="w-8 h-0.5 bg-dark-700"></div>
                </div>
              </>
            )}

            {/* Gateway/Router */}
            <div className="flex flex-col items-center min-w-[100px]">
              <div className="w-16 h-16 rounded-xl bg-orange-500/20 flex items-center justify-center mb-2">
                <Router className="w-8 h-8 text-orange-400" />
              </div>
              <p className="text-sm font-medium text-white">Gateway</p>
              <p className="text-xs text-dark-400 font-mono">{data.gateway || '--'}</p>
            </div>

            {/* External Router (Double NAT) */}
            {data.externalRouter && (
              <>
                <div className="flex items-center">
                  <div className="w-8 h-0.5 bg-dark-700"></div>
                  <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                  <div className="w-8 h-0.5 bg-dark-700"></div>
                </div>

                <div className="flex flex-col items-center min-w-[100px]">
                  <div className="w-16 h-16 rounded-xl bg-yellow-500/20 flex items-center justify-center mb-2">
                    <Router className="w-8 h-8 text-yellow-400" />
                  </div>
                  <p className="text-sm font-medium text-white">ISP Router</p>
                  <p className="text-xs text-dark-400 font-mono">{data.externalRouter}</p>
                </div>
              </>
            )}

            {/* Connection Line */}
            <div className="flex items-center">
              <div className="w-8 h-0.5 bg-dark-700"></div>
              <div className={`w-3 h-3 rounded-full ${data.latency ? 'bg-green-500' : 'bg-dark-600'}`}></div>
              <div className="w-8 h-0.5 bg-dark-700"></div>
            </div>

            {/* Internet */}
            <div className="flex flex-col items-center min-w-[100px]">
              <div className="w-16 h-16 rounded-xl bg-purple-500/20 flex items-center justify-center mb-2">
                <Globe className="w-8 h-8 text-purple-400" />
              </div>
              <p className="text-sm font-medium text-white">Internet</p>
              <p className="text-xs text-dark-400 font-mono">{data.publicIp || (data.latency ? `${data.latency.toFixed(0)}ms` : '--')}</p>
            </div>
          </div>
        </div>
      </motion.div>
    </motion.div>
  )
}
