import { motion, AnimatePresence } from 'framer-motion'
import { X, AlertCircle, CheckCircle, Info, AlertTriangle } from 'lucide-react'

/**
 * Toast notification types
 */
const TOAST_TYPES = {
  success: {
    icon: CheckCircle,
    className: 'bg-green-500/20 border-green-500/30 text-green-400',
    iconClass: 'text-green-400',
  },
  error: {
    icon: AlertCircle,
    className: 'bg-red-500/20 border-red-500/30 text-red-400',
    iconClass: 'text-red-400',
  },
  warning: {
    icon: AlertTriangle,
    className: 'bg-yellow-500/20 border-yellow-500/30 text-yellow-400',
    iconClass: 'text-yellow-400',
  },
  info: {
    icon: Info,
    className: 'bg-blue-500/20 border-blue-500/30 text-blue-400',
    iconClass: 'text-blue-400',
  },
}

/**
 * Toast notification component
 */
export function Toast({ type = 'info', message, onClose, duration = 5000 }) {
  const config = TOAST_TYPES[type] || TOAST_TYPES.info
  const Icon = config.icon

  return (
    <motion.div
      initial={{ opacity: 0, y: -20, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -20, scale: 0.95 }}
      className={`flex items-center gap-3 px-4 py-3 rounded-xl border ${config.className}`}
    >
      <Icon className={`w-5 h-5 flex-shrink-0 ${config.iconClass}`} />
      <span className="flex-1 text-sm">{message}</span>
      {onClose && (
        <button
          onClick={onClose}
          className="p-1 rounded-lg hover:bg-white/10 transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </motion.div>
  )
}

/**
 * Loading spinner component
 */
export function Spinner({ size = 'md', className = '' }) {
  const sizes = {
    sm: 'w-4 h-4',
    md: 'w-6 h-6',
    lg: 'w-8 h-8',
    xl: 'w-12 h-12',
  }

  return (
    <svg
      className={`animate-spin ${sizes[size]} ${className}`}
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  )
}

/**
 * Loading overlay component
 */
export function LoadingOverlay({ message = 'Loading...' }) {
  return (
    <div className="absolute inset-0 bg-dark-900/80 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="flex flex-col items-center gap-4">
        <Spinner size="lg" className="text-primary-400" />
        <span className="text-dark-300">{message}</span>
      </div>
    </div>
  )
}

/**
 * Empty state component
 */
export function EmptyState({ icon: Icon, title, description, action }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      {Icon && (
        <div className="w-16 h-16 rounded-2xl bg-dark-800/50 flex items-center justify-center mb-4">
          <Icon className="w-8 h-8 text-dark-400" />
        </div>
      )}
      <h3 className="text-lg font-semibold text-white mb-2">{title}</h3>
      {description && (
        <p className="text-dark-400 max-w-md mb-4">{description}</p>
      )}
      {action}
    </div>
  )
}

/**
 * Badge component
 */
export function Badge({ children, variant = 'default', size = 'md' }) {
  const variants = {
    default: 'bg-dark-700 text-dark-300',
    primary: 'bg-primary-500/20 text-primary-400',
    success: 'bg-green-500/20 text-green-400',
    warning: 'bg-yellow-500/20 text-yellow-400',
    error: 'bg-red-500/20 text-red-400',
    cyan: 'bg-cyan-500/20 text-cyan-400',
    purple: 'bg-purple-500/20 text-purple-400',
    orange: 'bg-orange-500/20 text-orange-400',
  }

  const sizes = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-xs',
    lg: 'px-3 py-1.5 text-sm',
  }

  return (
    <span className={`inline-flex items-center gap-1 rounded-full font-medium ${variants[variant]} ${sizes[size]}`}>
      {children}
    </span>
  )
}

/**
 * Card component
 */
export function Card({ children, className = '', gradient = null, padding = 'p-6' }) {
  const gradients = {
    blue: 'gradient-blue',
    cyan: 'gradient-cyan',
    purple: 'gradient-purple',
    orange: 'gradient-orange',
  }

  return (
    <div className={`glass-card ${gradient ? gradients[gradient] : ''} ${padding} ${className}`}>
      {children}
    </div>
  )
}

/**
 * Card header component
 */
export function CardHeader({ icon: Icon, title, badge, actions, iconColor = 'text-primary-400', iconBg = 'bg-primary-500/20' }) {
  return (
    <div className="flex items-center justify-between mb-6">
      <div className="flex items-center gap-3">
        {Icon && (
          <div className={`p-2 rounded-lg ${iconBg}`}>
            <Icon className={`w-5 h-5 ${iconColor}`} />
          </div>
        )}
        <h3 className="text-lg font-semibold text-white">{title}</h3>
        {badge}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  )
}

/**
 * Button component
 */
export function Button({
  children,
  variant = 'primary',
  size = 'md',
  disabled = false,
  loading = false,
  icon: Icon,
  onClick,
  className = '',
  type = 'button',
}) {
  const variants = {
    primary: 'btn-primary',
    secondary: 'btn-secondary',
    danger: 'bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30',
    ghost: 'hover:bg-dark-800/50 text-dark-300 hover:text-white',
  }

  const sizes = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2',
    lg: 'px-6 py-3 text-lg',
  }

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled || loading}
      className={`
        inline-flex items-center justify-center gap-2 rounded-xl font-medium transition-all
        ${variants[variant]}
        ${sizes[size]}
        ${disabled || loading ? 'opacity-50 cursor-not-allowed' : ''}
        ${className}
      `}
    >
      {loading ? (
        <Spinner size="sm" />
      ) : Icon ? (
        <Icon className="w-4 h-4" />
      ) : null}
      {children}
    </button>
  )
}

/**
 * Tabs component
 */
export function Tabs({ tabs, activeTab, onChange }) {
  return (
    <div className="flex gap-2 p-1 bg-dark-900/50 rounded-xl w-fit">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          onClick={() => onChange(tab.id)}
          className={`
            px-4 py-2 rounded-lg text-sm font-medium transition-all
            ${activeTab === tab.id
              ? 'bg-primary-500/20 text-primary-400'
              : 'text-dark-400 hover:text-white'}
          `}
        >
          {tab.label}
          {tab.count !== undefined && (
            <span className="ml-2 text-xs opacity-70">({tab.count})</span>
          )}
        </button>
      ))}
    </div>
  )
}

/**
 * Progress bar component
 */
export function ProgressBar({ value, max = 100, showLabel = false, size = 'md', color = 'primary' }) {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100))
  
  const sizes = {
    sm: 'h-1',
    md: 'h-2',
    lg: 'h-3',
  }

  const colors = {
    primary: 'bg-primary-500',
    success: 'bg-green-500',
    warning: 'bg-yellow-500',
    error: 'bg-red-500',
  }

  return (
    <div className="w-full">
      <div className={`w-full bg-dark-800 rounded-full overflow-hidden ${sizes[size]}`}>
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${percentage}%` }}
          transition={{ duration: 0.5, ease: 'easeOut' }}
          className={`h-full ${colors[color]} rounded-full`}
        />
      </div>
      {showLabel && (
        <div className="flex justify-between mt-1 text-xs text-dark-400">
          <span>{value}</span>
          <span>{max}</span>
        </div>
      )}
    </div>
  )
}

/**
 * Skeleton loading component
 */
export function Skeleton({ className = '', variant = 'text' }) {
  const variants = {
    text: 'h-4 w-full',
    title: 'h-6 w-3/4',
    circle: 'h-10 w-10 rounded-full',
    card: 'h-32 w-full rounded-xl',
    image: 'h-48 w-full rounded-xl',
  }

  return (
    <div
      className={`animate-pulse bg-dark-800/50 rounded ${variants[variant]} ${className}`}
    />
  )
}

/**
 * Stat item component for displaying key-value pairs
 */
export function StatItem({ label, value, icon: Icon, trend, className = '' }) {
  return (
    <div className={`flex items-center justify-between py-2 ${className}`}>
      <div className="flex items-center gap-2">
        {Icon && <Icon className="w-4 h-4 text-dark-400" />}
        <span className="text-sm text-dark-400">{label}</span>
      </div>
      <div className="flex items-center gap-2">
        <span className="font-semibold text-white">{value}</span>
        {trend && (
          <span className={`text-xs ${trend > 0 ? 'text-green-400' : 'text-red-400'}`}>
            {trend > 0 ? '+' : ''}{trend}%
          </span>
        )}
      </div>
    </div>
  )
}

export default {
  Toast,
  Spinner,
  LoadingOverlay,
  EmptyState,
  Badge,
  Card,
  CardHeader,
  Button,
  Tabs,
  ProgressBar,
  Skeleton,
  StatItem,
}
