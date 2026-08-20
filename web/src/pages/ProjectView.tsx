import React, { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import axios from 'axios'
import { Key, Webhook, ArrowLeft, CheckCircle2, Terminal } from 'lucide-react'

interface Project {
  id: string
  name: string
  created_at: string
}

export default function ProjectView() {
  const { id } = useParams()
  const [project, setProject] = useState<Project | null>(null)
  
  // API Key Form State
  const [provider, setProvider] = useState('openai')
  const [apiKey, setApiKey] = useState('')
  const [keyStatus, setKeyStatus] = useState<string | null>(null)

  const [logs, setLogs] = useState<string[]>([])
  const logsEndRef = React.useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchProject()

    // Connect to WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host // In dev, this is Vite (5173), so proxy handles it
    const ws = new WebSocket(`${protocol}//${host}/api/v1/projects/${id}/logs/stream`)

    ws.onmessage = (event) => {
      setLogs((prev) => [...prev, event.data])
    }

    return () => {
      ws.close()
    }
  }, [id])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const fetchProject = async () => {
    try {
      const res = await axios.get(`/api/v1/projects/${id}`)
      setProject(res.data)
    } catch (err) {
      console.error(err)
    }
  }

  const handleSaveKey = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!apiKey.trim()) return
    try {
      await axios.post(`/api/v1/projects/${id}/credentials`, {
        provider,
        api_key: apiKey
      })
      setKeyStatus('Key saved successfully!')
      setApiKey('')
      setTimeout(() => setKeyStatus(null), 3000)
    } catch (err) {
      setKeyStatus('Failed to save key.')
    }
  }

  if (!project) return <div className="p-12 text-center text-slate-500">Loading project...</div>

  // Generate the webhook URL using window.location.origin
  // For localtunnel/pinggy, they might need to use their tunnel url, but for local testing this is fine.
  const webhookUrl = `${window.location.origin}/api/v1/projects/${id}/webhooks/github`

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <Link to="/" className="inline-flex items-center text-sm font-medium text-slate-500 hover:text-slate-800">
        <ArrowLeft size={16} className="mr-1" /> Back to Projects
      </Link>

      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold text-slate-800">{project.name}</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        
        {/* Webhook Configuration Card */}
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-6 space-y-4">
          <div className="flex items-center gap-2 border-b border-slate-100 pb-4">
            <Webhook className="text-purple-600" />
            <h2 className="text-lg font-semibold text-slate-800">GitHub Webhook</h2>
          </div>
          <p className="text-sm text-slate-600">
            Configure this URL in your GitHub repository settings under <strong>Webhooks</strong> to trigger the pipeline automatically.
          </p>
          <div className="bg-slate-50 p-3 rounded-lg border border-slate-200 overflow-x-auto">
            <code className="text-sm text-purple-700 whitespace-nowrap">{webhookUrl}</code>
          </div>
          <p className="text-xs text-slate-500">
            Content type: <code className="bg-slate-100 px-1 py-0.5 rounded">application/json</code>
          </p>
        </div>

        {/* AI Credentials Card */}
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-6 space-y-4">
          <div className="flex items-center gap-2 border-b border-slate-100 pb-4">
            <Key className="text-amber-500" />
            <h2 className="text-lg font-semibold text-slate-800">AI Credentials</h2>
          </div>
          <p className="text-sm text-slate-600">
            Add an API key to enable AI-powered vulnerability scanning and code reviews.
          </p>
          
          <form onSubmit={handleSaveKey} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Provider</label>
              <select 
                value={provider} 
                onChange={e => setProvider(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              >
                <option value="openai">OpenAI</option>
                <option value="claude">Anthropic (Claude)</option>
                <option value="bedrock">AWS Bedrock</option>
              </select>
            </div>
            
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">API Key</label>
              <input 
                type="password" 
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                placeholder="sk-..."
                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              />
            </div>

            <button 
              type="submit"
              className="w-full bg-slate-800 hover:bg-slate-900 text-white px-4 py-2 rounded-lg font-medium text-sm transition-colors"
            >
              Save API Key
            </button>
            
            {keyStatus && (
              <div className="flex items-center gap-2 text-sm text-emerald-600 mt-2">
                <CheckCircle2 size={16} /> {keyStatus}
              </div>
            )}
          </form>
        </div>

      </div>
      
      {/* Live Pipeline Logs */}
      <div className="bg-slate-950 rounded-xl shadow-xl overflow-hidden border border-slate-800">
        <div className="flex items-center px-4 py-3 bg-slate-900 border-b border-slate-800">
          <Terminal size={18} className="text-emerald-400 mr-2" />
          <h2 className="text-sm font-semibold text-slate-200">Live Pipeline Logs</h2>
          <div className="ml-auto flex gap-2">
            <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
            <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
            <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
          </div>
        </div>
        
        <div className="p-4 h-96 overflow-y-auto font-mono text-xs text-slate-300 space-y-1">
          {logs.length === 0 ? (
            <div className="text-slate-600 italic">Waiting for pipeline execution...</div>
          ) : (
            logs.map((log, i) => (
              <div key={i} className="whitespace-pre-wrap">{log}</div>
            ))
          )}
          <div ref={logsEndRef} />
        </div>
      </div>

    </div>
  )
}
