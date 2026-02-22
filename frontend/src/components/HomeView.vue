<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

interface TorrentStats {
  seeders: number
  leechers: number
  completed: number
}

interface Torrent {
  id: string
  name: string
  info_hash: string
  total_size: number
  file_count: number
  magnet_link: string
  piece_length: number
  created_at: string
}

interface TorrentWithStats extends Torrent {
  stats: TorrentStats
}

const torrents = ref<TorrentWithStats[]>([])
const loading = ref(true)
const error = ref('')
const searchQuery = ref('')

const fetchTorrents = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('/api/v1/torrents')
    if (!res.ok) throw new Error('Failed to fetch data')
    const data = await res.json()
    torrents.value = data.data || []
  } catch (err: any) {
    error.value = err.message || 'An error occurred while fetching models.'
    console.error(err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchTorrents()
})

const formatBytes = (bytes: number, decimals = 2) => {
  if (!+bytes) return '0 Bytes'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

const copyMagnet = async (link: string) => {
  try {
    await navigator.clipboard.writeText(link)
    // Here you would typically show a toast notification
    alert('Magnet link copied to clipboard!')
  } catch (err) {
    console.error('Failed to copy text: ', err)
  }
}

// Simple client-side search
const filteredTorrents = computed(() => {
  if (!searchQuery.value) return torrents.value
  const query = searchQuery.value.toLowerCase()
  return torrents.value.filter(t => t.name.toLowerCase().includes(query))
})
</script>

<template>
  <div class="models-view">
    <div class="page-header">
      <div>
        <h1>LLM Weights & Checkpoints</h1>
        <p class="subtitle">Distributed via optimized BEP-0023 Protocol</p>
      </div>
      
      <div class="search-box glass-panel">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <input 
          type="text" 
          v-model="searchQuery" 
          placeholder="Search models... (e.g. Llama-3)"
        >
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="state-container glass-panel">
      <div class="loader"></div>
      <p>Synchronizing with Tracker node...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="state-container error-state glass-panel">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--danger)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="8" x2="12" y2="12"></line>
        <line x1="12" y1="16" x2="12.01" y2="16"></line>
      </svg>
      <p>{{ error }}</p>
      <button class="btn btn-primary" @click="fetchTorrents" style="margin-top: 1rem;">Retry connection</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredTorrents.length === 0" class="state-container glass-panel">
      <p v-if="searchQuery">No models found matching "{{ searchQuery }}".</p>
      <p v-else>No models have been published to this tracker yet.</p>
    </div>

    <!-- Data Table -->
    <div v-else class="table-container glass-panel">
      <table>
        <thead>
          <tr>
            <th class="col-name">Model Name</th>
            <th class="col-size">Size</th>
            <th class="col-files">Files</th>
            <th class="col-stats">Swarm Health</th>
            <th class="col-action">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="torrent in filteredTorrents" :key="torrent.id">
            <td class="col-name">
              <div class="model-info">
                <span class="model-title">{{ torrent.name }}</span>
                <span class="model-hash">{{ torrent.info_hash.substring(0, 8) }}...{{ torrent.info_hash.substring(32) }}</span>
              </div>
            </td>
            <td class="col-size">{{ formatBytes(torrent.total_size) }}</td>
            <td class="col-files">{{ torrent.file_count }}</td>
            <td class="col-stats">
              <div class="stats-group">
                <span class="badge badge-success" title="Seeders">S: {{ torrent.stats.seeders }}</span>
                <span class="badge badge-warning" title="Leechers">L: {{ torrent.stats.leechers }}</span>
                <span class="completed" v-if="torrent.stats.completed > 0">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                  {{ torrent.stats.completed }}
                </span>
              </div>
            </td>
            <td class="col-action">
              <button class="action-btn" @click="copyMagnet(torrent.magnet_link)" title="Copy Magnet Link">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
                  <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
                </svg>
                Copy Magnet
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 2rem;
  flex-wrap: wrap;
  gap: 1rem;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  min-width: 300px;
}

.search-box input {
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 0.95rem;
  width: 100%;
  outline: none;
}

.search-box input::placeholder {
  color: var(--text-tertiary);
}

.search-box svg {
  color: var(--text-secondary);
}

/* State Containers */
.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  text-align: center;
  color: var(--text-secondary);
}

.loader {
  border: 3px solid var(--surface-border);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Table Styles */
.table-container {
  padding: 0;
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

th {
  padding: 1.25rem 1.5rem;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--surface-border);
  background: rgba(0,0,0,0.2);
}

td {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid rgba(255,255,255,0.03);
  vertical-align: middle;
}

tr:last-child td {
  border-bottom: none;
}

tr:hover td {
  background: rgba(255,255,255,0.02);
}

/* Columns */
.col-size, .col-files {
  white-space: nowrap;
  color: var(--text-secondary);
}

.model-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.model-title {
  font-weight: 600;
  font-size: 1rem;
  color: var(--text-primary);
}

.model-hash {
  font-family: monospace;
  font-size: 0.75rem;
  color: var(--text-tertiary);
}

.stats-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.completed {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

/* Actions */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.875rem;
  background: rgba(255,255,255,0.05);
  border: 1px solid var(--surface-border);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 0.85rem;
}

.action-btn:hover {
  background: rgba(255,255,255,0.1);
  color: var(--text-primary);
  border-color: rgba(255,255,255,0.15);
}
</style>
