import { BrowserRouter, Routes, Route, Link } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import ProjectView from './pages/ProjectView'

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-slate-50 flex flex-col">
        {/* Top Navbar */}
        <header className="bg-white border-b border-slate-200 px-6 py-4 flex items-center justify-between shadow-sm">
          <Link to="/" className="flex items-center gap-3 text-xl font-bold text-slate-800">
            <img src="/logo.jpg" alt="Rune Logo" className="w-8 h-8 rounded-md" />
            Rune
          </Link>
          <div className="text-sm text-slate-500 font-medium">Control Plane</div>
        </header>

        {/* Main Content Area */}
        <main className="flex-1 max-w-7xl w-full mx-auto p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/projects/:id" element={<ProjectView />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}

export default App
