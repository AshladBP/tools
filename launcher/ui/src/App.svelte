<script lang="ts">
  import { onMount } from 'svelte';
  import * as App from './lib/wailsjs/go/main/App';
  import { EventsOn } from './lib/wailsjs/runtime/runtime';
  import { main } from './lib/wailsjs/go/models';

  type Status = main.Status;
  type Config = main.Config;
  type WatcherStatus = main.WatcherStatus;
  type PortStatus = main.PortStatus;

  interface LogEntry {
    source: string;
    message: string;
  }

  let status: Status = $state({
    backend: 'stopped',
    frontend: 'stopped',
    backendPid: 0,
    frontendPid: 0,
    mtoolsExists: false,
    libraryPath: '',
    frontendPort: '7750',
    backendPort: '7754',
  });

  let logs: string[] = $state([]);
  let showLogs = $state(false);
  let showSettings = $state(false);
  let showEmbed = $state(true);
  let loading = $state(false);
  let error = $state('');
  let frontendUrl = $state('');
  let iframeKey = $state(0);

  let config: Config = $state({
    libraryPath: '',
    frontendPort: '7750',
  });

  let deps: Record<string, boolean> = $state({});

  // Watcher status
  let watcherStatus: WatcherStatus = $state({ available: false, enabled: false });
  let watcherLoading = $state(false);

  // Ports status for kill buttons
  let portsStatus: PortStatus[] = $state([]);
  let portsLoading = $state(false);

  onMount(async () => {
    // Wait for Wails to be ready
    await new Promise(resolve => setTimeout(resolve, 100));

    try {
      // Load initial data
      await refreshStatus();
      config = await App.GetConfig();
      deps = await App.CheckDependencies();
      frontendUrl = await App.GetFrontendURL();

      // Subscribe to log events
      EventsOn('log', (data: LogEntry) => {
        logs.push(`[${data.source.toUpperCase()}] ${data.message}`);
        if (logs.length > 200) {
          logs = logs.slice(-200);
        }
      });

      // Subscribe to status changes
      EventsOn('statusChange', (newStatus: Status) => {
        status = newStatus;
      });

      // Poll status every 2 seconds
      setInterval(refreshStatus, 2000);
    } catch (e) {
      console.error('Failed to initialize:', e);
      error = 'Failed to connect to backend';
    }
  });

  async function refreshStatus() {
    try {
      status = await App.GetStatus();
      frontendUrl = await App.GetFrontendURL();
    } catch (e) {
      console.error('Failed to get status:', e);
    }
  }

  async function startAll() {
    loading = true;
    error = '';
    try {
      await App.StartAll();
      await refreshStatus();
      // Reload iframe after a delay
      setTimeout(() => iframeKey++, 3000);
    } catch (e: any) {
      error = e.message || String(e);
    }
    loading = false;
  }

  async function stopAll() {
    loading = true;
    error = '';
    try {
      await App.StopAll();
      await refreshStatus();
    } catch (e: any) {
      error = e.message || String(e);
    }
    loading = false;
  }

  async function restartAll() {
    loading = true;
    error = '';
    try {
      await App.RestartAll();
      await refreshStatus();
      setTimeout(() => iframeKey++, 3000);
    } catch (e: any) {
      error = e.message || String(e);
    }
    loading = false;
  }

  let selectingFolder = $state(false);

  async function selectFolder() {
    error = '';
    selectingFolder = true;
    try {
      const path = await App.SelectLibraryFolder();
      if (path) {
        config.libraryPath = path;
        await refreshStatus();
      }
    } catch (e: any) {
      console.error('selectFolder error:', e);
      error = e.message || String(e);
    } finally {
      selectingFolder = false;
    }
  }

  async function saveConfig() {
    try {
      await App.SetConfig(config);
      showSettings = false;
      await refreshStatus();
    } catch (e: any) {
      error = e.message || String(e);
    }
  }

  async function openInBrowser() {
    try {
      await App.OpenMTools();
    } catch (e: any) {
      error = e.message || String(e);
    }
  }

  function clearLogs() {
    App.ClearLogs('all');
    logs = [];
  }

  function reloadIframe() {
    iframeKey++;
  }

  // Watcher functions
  async function refreshWatcherStatus() {
    if (status.backend !== 'running') {
      watcherStatus = { available: false, enabled: false };
      return;
    }
    try {
      watcherStatus = await App.GetWatcherStatus();
    } catch (e) {
      console.error('Failed to get watcher status:', e);
      watcherStatus = { available: false, enabled: false };
    }
  }

  async function toggleWatcher() {
    watcherLoading = true;
    try {
      await App.SetWatcherEnabled(!watcherStatus.enabled);
      await refreshWatcherStatus();
    } catch (e: any) {
      error = e.message || String(e);
    }
    watcherLoading = false;
  }

  // Port management functions
  async function checkPorts() {
    portsLoading = true;
    try {
      portsStatus = await App.GetPortsStatus();
    } catch (e: any) {
      console.error('Failed to check ports:', e);
      portsStatus = [];
    }
    portsLoading = false;
  }

  async function killPortProcess(port: string) {
    portsLoading = true;
    error = '';
    try {
      await App.KillProcessOnPort(port);
      await checkPorts();
    } catch (e: any) {
      error = e.message || String(e);
    }
    portsLoading = false;
  }

  async function killAllPortProcesses() {
    portsLoading = true;
    error = '';
    try {
      await App.KillPortProcesses();
      await checkPorts();
    } catch (e: any) {
      error = e.message || String(e);
    }
    portsLoading = false;
  }

  $effect(() => {
    // Auto-show embed when frontend is running
    if (status.frontend === 'running' && !showSettings && !showLogs) {
      showEmbed = true;
    }
  });

  // Refresh watcher status when backend status changes
  $effect(() => {
    if (status.backend === 'running') {
      refreshWatcherStatus();
    } else {
      watcherStatus = { available: false, enabled: false };
    }
  });

  // Check ports when services are stopped
  $effect(() => {
    if (status.backend === 'stopped' && status.frontend === 'stopped') {
      checkPorts();
    }
  });
</script>

<div class="app">
  <!-- Header -->
  <header class="header">
    <div class="header-left">
      <h1>MTools Launcher</h1>
    </div>

    <div class="header-center">
      <div class="status-bar">
        <div class="status-item">
          <span class="status-dot" class:running={status.backend === 'running'} class:stopped={status.backend === 'stopped'}></span>
          <span>Backend</span>
          {#if status.backendPid > 0}
            <span class="pid">:{status.backendPort}</span>
          {/if}
        </div>
        <div class="status-item">
          <span class="status-dot" class:running={status.frontend === 'running'} class:stopped={status.frontend === 'stopped'}></span>
          <span>Frontend</span>
          {#if status.frontendPid > 0}
            <span class="pid">:{status.frontendPort}</span>
          {/if}
        </div>
      </div>
    </div>

    <div class="header-right">
      <button class="secondary" onclick={() => showSettings = !showSettings} aria-label="Settings">
        ⚙️
      </button>
      <button class="secondary" onclick={() => { showLogs = !showLogs; showEmbed = !showLogs; }}>
        📋 Logs
      </button>
    </div>
  </header>

  <!-- Controls -->
  <div class="controls">
    <div class="control-buttons">
      {#if status.backend === 'stopped' && status.frontend === 'stopped'}
        <button class="primary" onclick={startAll} disabled={loading || !status.libraryPath}>
          ▶️ Start All
        </button>
        <button class="secondary" onclick={selectFolder} disabled={selectingFolder}>
          {selectingFolder ? '...' : '📂 Change Dir'}
        </button>
        <label class="autoload-toggle">
          <input
            type="checkbox"
            bind:checked={config.autoLoadBooks}
            onchange={saveConfig}
          />
          <span>Auto-load books</span>
        </label>
      {:else if status.backend === 'running' && status.frontend === 'running'}
        <button class="danger" onclick={stopAll} disabled={loading}>
          ⏹️ Stop All
        </button>
        <button class="secondary" onclick={restartAll} disabled={loading}>
          🔄 Restart
        </button>
      {:else}
        <button class="primary" onclick={startAll} disabled={loading}>
          ▶️ Start All
        </button>
        <button class="danger" onclick={stopAll} disabled={loading}>
          ⏹️ Stop All
        </button>
      {/if}

      {#if status.frontend === 'running'}
        <button class="secondary" onclick={openInBrowser}>
          🌐 Open in Browser
        </button>
        <button class="secondary" onclick={reloadIframe}>
          🔄 Reload
        </button>
      {/if}

      <!-- Watcher toggle when backend is running -->
      {#if status.backend === 'running' && watcherStatus.available}
        <label class="watcher-toggle">
          <input
            type="checkbox"
            checked={watcherStatus.enabled}
            onchange={toggleWatcher}
            disabled={watcherLoading}
          />
          <span>Auto-reload LUT</span>
        </label>
      {/if}
    </div>

    <!-- Port conflicts warning -->
    {#if status.backend === 'stopped' && status.frontend === 'stopped'}
      {#if portsStatus.some(p => p.inUse)}
        <div class="warning-message">
          ⚠️ Ports in use:
          {#each portsStatus.filter(p => p.inUse) as port}
            <span class="port-badge">
              {port.port}
              <button class="kill-btn" onclick={() => killPortProcess(port.port)} disabled={portsLoading}>
                ✕
              </button>
            </span>
          {/each}
          <button class="secondary small" onclick={killAllPortProcesses} disabled={portsLoading}>
            Kill All
          </button>
        </div>
      {/if}
    {/if}

    {#if error}
      <div class="error-message">{error}</div>
    {/if}

    {#if !status.libraryPath}
      <div class="warning-message">
        ⚠️ Library path not set.
        <button class="secondary" onclick={selectFolder} disabled={selectingFolder}>
          {selectingFolder ? '...' : 'Select Folder'}
        </button>
      </div>
    {/if}
  </div>

  <!-- Main Content -->
  <main class="main">
    {#if showSettings}
      <div class="settings-panel">
        <h2>Settings</h2>

        <div class="setting-group">
          <label for="library-path">Library Path</label>
          <div class="input-with-button">
            <input id="library-path" type="text" bind:value={config.libraryPath} placeholder="/path/to/library" />
            <button class="secondary" onclick={selectFolder} disabled={selectingFolder}>
              {selectingFolder ? '...' : 'Browse'}
            </button>
          </div>
        </div>

        <div class="setting-group">
          <label for="frontend-port">Frontend Port</label>
          <input id="frontend-port" type="text" bind:value={config.frontendPort} placeholder="7750" />
        </div>

        <div class="setting-group">
          <label class="checkbox-setting">
            <input type="checkbox" bind:checked={config.autoLoadBooks} />
            <span>Auto-load event books at startup</span>
          </label>
          <p class="setting-hint">When disabled, books won't load automatically to prevent high CPU usage. You can manually start loading from the frontend.</p>
        </div>

        <div class="setting-group">
          <span class="setting-label">Dependencies</span>
          <div class="deps-list">
            {#each Object.entries(deps) as [name, installed]}
              <div class="dep-item">
                <span class="status-dot" class:running={installed} class:stopped={!installed}></span>
                <span>{name}</span>
                <span class="dep-status">{installed ? '✓' : '✗'}</span>
              </div>
            {/each}
          </div>
        </div>

        <div class="setting-actions">
          <button class="primary" onclick={saveConfig}>Save</button>
          <button class="secondary" onclick={() => showSettings = false}>Cancel</button>
        </div>
      </div>
    {:else if showLogs}
      <div class="logs-panel">
        <div class="logs-header">
          <h2>Logs</h2>
          <button class="secondary" onclick={clearLogs}>Clear</button>
        </div>
        <div class="logs-content">
          {#each logs as log}
            <div class="log-line" class:backend={log.includes('[BACKEND]')} class:frontend={log.includes('[FRONTEND]')}>
              {log}
            </div>
          {/each}
          {#if logs.length === 0}
            <div class="logs-empty">No logs yet. Start the services to see output.</div>
          {/if}
        </div>
      </div>
    {:else if showEmbed && status.frontend === 'running'}
      <div class="embed-container">
        {#key iframeKey}
          <iframe
            src={frontendUrl}
            title="MTools"
            sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
          ></iframe>
        {/key}
      </div>
    {:else}
      <div class="welcome">
        <div class="welcome-content">
          <h2>Welcome to MTools Launcher</h2>
          <p>Start the services to begin using MTools.</p>

          {#if !status.libraryPath}
            <p class="setup-hint">First, select your library folder in settings.</p>
          {:else}
            <button class="primary large" onclick={startAll} disabled={loading}>
              ▶️ Start All Services
            </button>
          {/if}
        </div>
      </div>
    {/if}
  </main>
</div>

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--bg-primary);
  }

  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
    gap: 20px;
  }

  .header-left h1 {
    font-size: 18px;
    font-weight: 600;
  }

  .header-center {
    flex: 1;
    display: flex;
    justify-content: center;
  }

  .header-right {
    display: flex;
    gap: 8px;
  }

  .status-bar {
    display: flex;
    gap: 24px;
  }

  .status-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
  }

  .pid {
    color: var(--text-secondary);
    font-size: 12px;
  }

  .controls {
    padding: 12px 20px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
  }

  .control-buttons {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .error-message {
    margin-top: 12px;
    padding: 8px 12px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid var(--error);
    border-radius: 6px;
    color: var(--error);
    font-size: 14px;
  }

  .warning-message {
    margin-top: 12px;
    padding: 8px 12px;
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid var(--warning);
    border-radius: 6px;
    color: var(--warning);
    font-size: 14px;
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .main {
    flex: 1;
    overflow: hidden;
    position: relative;
  }

  /* Settings */
  .settings-panel {
    padding: 24px;
    max-width: 600px;
    margin: 0 auto;
  }

  .settings-panel h2 {
    margin-bottom: 24px;
  }

  .setting-group {
    margin-bottom: 20px;
  }

  .setting-group label,
  .setting-label {
    display: block;
    margin-bottom: 8px;
    font-size: 14px;
    color: var(--text-secondary);
  }

  .setting-group input {
    width: 100%;
  }

  .input-with-button {
    display: flex;
    gap: 8px;
  }

  .input-with-button input {
    flex: 1;
  }

  .deps-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 6px;
  }

  .dep-item {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .dep-status {
    margin-left: auto;
    font-size: 12px;
  }

  .setting-actions {
    display: flex;
    gap: 8px;
    margin-top: 24px;
  }

  /* Logs */
  .logs-panel {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .logs-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border);
  }

  .logs-header h2 {
    font-size: 16px;
  }

  .logs-content {
    flex: 1;
    overflow-y: auto;
    padding: 12px 20px;
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
    font-size: 12px;
    line-height: 1.6;
  }

  .log-line {
    white-space: pre-wrap;
    word-break: break-all;
  }

  .log-line.backend {
    color: #60a5fa;
  }

  .log-line.frontend {
    color: #34d399;
  }

  .logs-empty {
    color: var(--text-secondary);
    text-align: center;
    padding: 40px;
  }

  /* Embed */
  .embed-container {
    height: 100%;
    width: 100%;
  }

  .embed-container iframe {
    width: 100%;
    height: 100%;
    border: none;
    background: white;
  }

  /* Welcome */
  .welcome {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .welcome-content {
    text-align: center;
  }

  .welcome-content h2 {
    margin-bottom: 12px;
  }

  .welcome-content p {
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .setup-hint {
    color: var(--warning) !important;
  }

  button.large {
    padding: 12px 32px;
    font-size: 16px;
    margin-top: 16px;
  }

  button.small {
    padding: 4px 8px;
    font-size: 12px;
  }

  /* Watcher toggle */
  .watcher-toggle,
  .autoload-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    background: var(--bg-tertiary);
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    user-select: none;
  }

  .watcher-toggle input,
  .autoload-toggle input {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .watcher-toggle:hover,
  .autoload-toggle:hover {
    background: var(--bg-hover);
  }

  /* Checkbox setting in settings panel */
  .checkbox-setting {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 14px;
  }

  .checkbox-setting input {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .setting-hint {
    margin-top: 6px;
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  /* Port badges */
  .port-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    background: rgba(239, 68, 68, 0.2);
    border-radius: 4px;
    font-family: monospace;
    font-size: 13px;
  }

  .kill-btn {
    padding: 0 4px;
    background: transparent;
    border: none;
    color: var(--error);
    cursor: pointer;
    font-size: 12px;
    line-height: 1;
  }

  .kill-btn:hover {
    color: #dc2626;
  }

  .kill-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
