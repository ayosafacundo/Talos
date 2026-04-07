<script>
  import { onMount } from 'svelte'
  import { EventsOn } from '../wailsjs/runtime/runtime'

  let socketURL = ''
  let packageMap = {}
  let eventLog = []
  let running = []

  const appApi = () => window?.go?.main?.App

  async function refresh() {
    const api = appApi()
    if (!api) {
      return
    }
    packageMap = await api.ListPackages()
    running = await api.RunningPackageIDs()
    socketURL = await api.HubSocketURL()
  }

  async function startPackage(id) {
    const api = appApi()
    if (!api) return
    await api.StartPackage(id)
    await refresh()
  }

  async function stopPackage(id) {
    const api = appApi()
    if (!api) return
    await api.StopPackage(id)
    await refresh()
  }

  onMount(async () => {
    await refresh()
    EventsOn('packages:event', async (evt) => {
      eventLog = [evt, ...eventLog].slice(0, 12)
      await refresh()
    })
  })
</script>

<main>
  <h1>Talos Phase 1</h1>
  <p>Hub endpoint: <code>{socketURL || 'starting...'}</code></p>

  <h2>Discovered Packages</h2>
  {#if Object.keys(packageMap).length === 0}
    <p>No valid manifests found in <code>/Packages</code>.</p>
  {:else}
    <ul>
      {#each Object.entries(packageMap) as [id, pkg]}
        <li>
          <strong>{pkg.manifest.name}</strong> <code>({id})</code>
          <div>
            <button on:click={() => startPackage(id)} disabled={running.includes(id)}>Start</button>
            <button on:click={() => stopPackage(id)} disabled={!running.includes(id)}>Stop</button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  <h2>Discovery Events</h2>
  <ul>
    {#each eventLog as evt}
      <li><code>{evt.type}</code> {evt.package_id || evt.error}</li>
    {/each}
  </ul>
</main>

<style>
  main {
    padding: 16px;
    text-align: left;
  }
  h1, h2 {
    margin: 0 0 8px;
  }
  h2 {
    margin-top: 16px;
    font-size: 16px;
  }
  ul {
    margin: 0;
    padding-left: 20px;
  }
  li {
    margin: 8px 0;
  }
  button {
    margin-right: 8px;
  }
</style>
