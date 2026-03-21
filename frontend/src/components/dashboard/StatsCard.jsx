import { motion } from 'framer-motion'

const gradientClasses = {
  blue: 'gradient-orange',
  cyan: 'gradient-cyan',
  purple: 'gradient-purple',
  orange: 'gradient-orange',
  teal: 'gradient-teal',
  pink: 'gradient-pink',
}

const iconClasses = {
  blue: 'bg-primary-500/20 text-primary-400',
  cyan: 'bg-cyan-500/20 text-cyan-400',
  purple: 'bg-purple-500/20 text-purple-400',
  orange: 'bg-orange-500/20 text-orange-400',
  teal: 'bg-teal-500/20 text-teal-400',
  pink: 'bg-pink-500/20 text-pink-400',
}

const badgeVariants = {
  success: 'bg-green-500/20 text-green-400 border-green-500/30',
  warning: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
  error: 'bg-red-500/20 text-red-400 border-red-500/30',
  cyan: 'bg-dark-500/20 text-dark-300 border-dark-500/30',
  purple: 'bg-dark-600/20 text-dark-300 border-dark-600/30',
  orange: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
}

export default function StatsCard({ 
  title, 
  gradient = 'blue', 
  icon: Icon, 
  badge, 
  children 
}) {
  return (
    <motion.div
      whileHover={{ y: -4, transition: { duration: 0.2 } }}
      className={`glass-card ${gradientClasses[gradient]} p-6`}
    >
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          {Icon && (
            <div className={`p-2 rounded-lg ${iconClasses[gradient] || iconClasses.blue}`}>
              <Icon className="w-5 h-5" />
            </div>
          )}
          <h3 className="text-lg font-semibold text-white">{title}</h3>
        </div>
        {badge && (
          <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium border ${badgeVariants[badge.variant] || badgeVariants.success}`}>
            <span className="relative flex h-1.5 w-1.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-current opacity-75"></span>
              <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-current"></span>
            </span>
            {badge.text}
          </span>
        )}
      </div>
      {children}
    </motion.div>
  )
}
