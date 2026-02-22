<script setup lang="ts">
import HomeView from './components/HomeView.vue'
</script>

<template>
  <div class="app-container">
    <header class="app-header">
      <div class="logo">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="var(--accent-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span>Model-PT</span>
      </div>
      <nav>
        <a href="#" class="active">Models</a>
        <a href="#">Publish</a>
      </nav>
    </header>

    <main>
      <HomeView />
    </main>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--surface-border);
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.025em;
}

nav {
  display: flex;
  gap: 1.5rem;
}

nav a {
  color: var(--text-secondary);
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s ease;
  position: relative;
  padding-bottom: 0.5rem;
}

nav a:hover, nav a.active {
  color: var(--text-primary);
}

nav a.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  width: 100%;
  height: 2px;
  background: var(--accent-primary);
  border-radius: 2px;
}
</style>
