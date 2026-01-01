import { useState, useEffect } from 'react'
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Sidebar from './components/layout/Sidebar'
import Header from './components/layout/Header'
import Dashboard from './pages/Dashboard'
import PluginManager from './pages/PluginManager'
import PluginPage from './pages/PluginPage'
import { useWebSocket } from './hooks/useWebSocket'
import { PluginProvider } from './context/PluginContext'

function AppContent() {
  const [darkMode, setDarkMode] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const { networkData, connected } = useWebSocket()

  useEffect(() => {
    // Load dark mode preference
    const savedMode = localStorage.getItem('darkMode')
    if (savedMode !== null) {
      setDarkMode(savedMode === 'true')
    }
  }, [])

  useEffect(() => {
    // Apply dark mode class
    if (darkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
    localStorage.setItem('darkMode', darkMode.toString())
  }, [darkMode])

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <Sidebar 
        isOpen={sidebarOpen} 
        onToggle={() => setSidebarOpen(!sidebarOpen)}
        darkMode={darkMode}
        onDarkModeToggle={() => setDarkMode(!darkMode)}
      />
      
      {/* Main Content */}
      <div className={`flex-1 flex flex-col transition-all duration-300 ${sidebarOpen ? 'ml-64' : 'ml-16'}`}>
        <Header 
          onMenuToggle={() => setSidebarOpen(!sidebarOpen)}
          connected={connected}
        />
        
        <main className="flex-1 p-6 overflow-auto">
          <Routes>
            <Route 
              path="/" 
              element={<Dashboard networkData={networkData} connected={connected} />} 
            />
            <Route 
              path="/dashboard" 
              element={<Dashboard networkData={networkData} connected={connected} />} 
            />
            <Route 
              path="/plugin-manager" 
              element={<PluginManager />} 
            />
            <Route 
              path="/plugin/:id" 
              element={<PluginPage />} 
            />
          </Routes>
        </main>
      </div>
    </div>
  )
}

function App() {
  return (
    <Router>
      <PluginProvider>
        <AppContent />
      </PluginProvider>
    </Router>
  )
}

export default App
