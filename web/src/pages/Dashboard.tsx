import React, { useEffect, useState } from 'react'
import axios from 'axios'
import { Link } from 'react-router-dom'
import { Plus, FolderGit2, ArrowRight } from 'lucide-react'

interface Project {
  id: string
  name: string
  created_at: string
}

export default function Dashboard() {
  const [projects, setProjects] = useState<Project[]>([])
  const [newProjectName, setNewProjectName] = useState('')
  const [isCreating, setIsCreating] = useState(false)

  useEffect(() => {
    fetchProjects()
  }, [])

  const fetchProjects = async () => {
    try {
      const res = await axios.get('/api/v1/projects')
      setProjects(res.data || [])
    } catch (err) {
      console.error("Failed to fetch projects", err)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newProjectName.trim()) return
    setIsCreating(true)
    try {
      await axios.post('/api/v1/projects', { name: newProjectName })
      setNewProjectName('')
      await fetchProjects()
    } catch (err) {
      console.error("Failed to create project", err)
    } finally {
      setIsCreating(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold text-slate-800">Projects</h1>
        
        <form onSubmit={handleCreate} className="flex gap-2">
          <input 
            type="text" 
            placeholder="New project name..."
            className="px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            value={newProjectName}
            onChange={(e) => setNewProjectName(e.target.value)}
            disabled={isCreating}
          />
          <button 
            type="submit" 
            disabled={isCreating}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 font-medium disabled:opacity-50"
          >
            <Plus size={18} />
            Create
          </button>
        </form>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {projects.length === 0 ? (
          <div className="col-span-full py-12 text-center text-slate-500 bg-white border border-slate-200 rounded-xl border-dashed">
            No projects found. Create one to get started!
          </div>
        ) : (
          projects.map(project => (
            <Link 
              to={`/projects/${project.id}`} 
              key={project.id}
              className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:shadow-md hover:border-blue-300 transition-all group flex flex-col justify-between h-40"
            >
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <div className="p-2 bg-blue-50 text-blue-600 rounded-lg">
                    <FolderGit2 size={24} />
                  </div>
                  <h3 className="text-lg font-semibold text-slate-800">{project.name}</h3>
                </div>
                <p className="text-xs text-slate-400 font-mono mt-4">ID: {project.id}</p>
              </div>
              <div className="flex items-center text-sm font-medium text-blue-600 opacity-0 group-hover:opacity-100 transition-opacity">
                View Project <ArrowRight size={16} className="ml-1" />
              </div>
            </Link>
          ))
        )}
      </div>
    </div>
  )
}
