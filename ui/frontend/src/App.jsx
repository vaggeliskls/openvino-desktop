import { useState, useEffect, useRef } from 'react'
import { GetConfig, SaveConfig, PrepareExport, PrepareOVMS } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

export default function App() {
  const [config, setConfig] = useState({ install_dir: '' })
  const [saved, setSaved] = useState(false)
  const [logs, setLogs] = useState([])
  const [running, setRunning] = useState(false)
  const [error, setError] = useState(null)
  const logsEndRef = useRef(null)

  useEffect(() => {
    GetConfig().then(setConfig)
    EventsOn('log', (line) => setLogs(prev => [...prev, line]))
  }, [])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const handleSave = async () => {
    await SaveConfig(config)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const run = (action) => {
    setLogs([])
    setError(null)
    setRunning(true)
    action()
      .then(() => setLogs(prev => [...prev, '--- Done ---']))
      .catch(err => setError(String(err)))
      .finally(() => setRunning(false))
  }

  return (
    <div className="app">
      <h1>OpenVINO Desk</h1>

      <section className="config-section">
        <h2>Configuration</h2>

        <div className="field">
          <label>Install Directory</label>
          <input
            value={config.install_dir}
            onChange={e => setConfig(c => ({ ...c, install_dir: e.target.value }))}
            placeholder="e.g. C:\Users\user\openvino-desk"
          />
          <small>Base directory where Python, venv and OVMS will be installed.</small>
        </div>

        <button onClick={handleSave} className="btn-save">
          {saved ? 'Saved!' : 'Save Configuration'}
        </button>
      </section>

      <section className="actions-section">
        <h2>Setup</h2>
        <div className="actions">
          <div className="action-card">
            <h3>Export Environment</h3>
            <p>Extracts bundled uv, installs Python 3.12, creates a virtual environment and installs all ML requirements.</p>
            <button
              className="btn-primary"
              disabled={running}
              onClick={() => run(PrepareExport)}
            >
              {running ? 'Running...' : 'Prepare Export'}
            </button>
          </div>

          <div className="action-card">
            <h3>OVMS Server</h3>
            <p>Downloads and extracts the OpenVINO Model Server for Windows.</p>
            <button
              className="btn-primary"
              disabled={running}
              onClick={() => run(PrepareOVMS)}
            >
              {running ? 'Running...' : 'Prepare OVMS'}
            </button>
          </div>
        </div>
      </section>

      {(logs.length > 0 || error) && (
        <section className="log-section">
          <h2>Output</h2>
          {error && <div className="error">{error}</div>}
          <div className="log-box">
            {logs.map((line, i) => (
              <div key={i} className={line.startsWith('---') ? 'log-done' : 'log-line'}>
                {line}
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        </section>
      )}
    </div>
  )
}
