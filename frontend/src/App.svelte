<script>
  import { onMount } from 'svelte'
  import { EventsOn } from '../wailsjs/runtime/runtime'

  let socketURL = ''
  let packageMap = {}
  let eventLog = []
  let running = []
  let permissionEvents = []
  let pendingPermissions = []
  let iframeEvents = []
  let selectedAppId = ''
  let iframeEvent = 'host:update'
  let iframePayload = '{"theme":"dark"}'

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

  async function grantFsExternal(id) {
    const api = appApi()
    if (!api) return
    await api.GrantPermission(id, 'fs:external')
  }

  async function decidePermission(appID, scope, allow) {
    const api = appApi()
    if (!api) return
    if (allow) {
      await api.GrantPermission(appID, scope)
    } else {
      await api.DenyPermission(appID, scope)
    }
    pendingPermissions = pendingPermissions.filter((p) => !(p.app_id === appID && p.scope === scope))
  }

  async function sendToIframe() {
    const api = appApi()
    if (!api || !selectedAppId) return
    await api.HostPostToIframe(selectedAppId, iframeEvent, iframePayload)
  }

  function onIframeLoad(event, appID) {
    const frame = event.currentTarget
    if (!frame || !frame.contentWindow) return

    frame.contentWindow.postMessage(
      {
        type: 'talos:bridge:from-host',
        app_id: appID,
        event: 'host:init',
        payload: '{"ready":true}'
      },
      '*'
    )
  }

  function getIframeSrc(pkg) {
    const entry = pkg?.manifest?.web_entry
    const dir = pkg?.dir_path
    if (!entry || !dir) {
      return ''
    }
    return encodeURI(`file://${dir}/${entry}`)
  }

  async function handleIframeSDKRequest(msg, sourceWindow) {
    const api = appApi()
    if (!api) return

    const requestId = String(msg.request_id || '')
    const appID = String(msg.app_id || '')
    const method = String(msg.method || '')
    const params = msg.params || {}

    const respond = (ok, result, error = '') => {
      sourceWindow?.postMessage(
        {
          type: 'talos:sdk:res',
          request_id: requestId,
          ok,
          result,
          error
        },
        '*'
      )
    }

    try {
      if (method === 'saveState') {
        await api.SaveAppStateBase64(appID, String(params.data_base64 || ''))
        respond(true, { ok: true })
        return
      }
      if (method === 'loadState') {
        const dataBase64 = await api.LoadAppStateBase64(appID)
        respond(true, { data_base64: dataBase64, found: dataBase64.length > 0 })
        return
      }
      if (method === 'requestPermission') {
        const result = await api.RequestPermissionDecision(appID, String(params.scope || ''), String(params.reason || ''))
        if (result?.error) {
          respond(false, null, String(result.error))
          return
        }
        respond(true, { granted: Boolean(result?.granted), message: String(result?.message || '') })
        return
      }
      if (method === 'resolvePath') {
        const resolvedPath = await api.ResolveScopedPath(appID, String(params.relative_path || ''))
        respond(true, { resolved_path: resolvedPath })
        return
      }
      if (method === 'sendMessage') {
        const payload = String(params.payload || '')
        const resp = await api.RouteMessage(appID, String(params.target_id || ''), String(params.type || 'sdk:message'), payload)
        respond(true, { payload: resp })
        return
      }
      if (method === 'broadcast') {
        const count = await api.BroadcastMessage(appID, String(params.type || 'sdk:broadcast'), String(params.payload || ''))
        respond(true, { recipients: count })
        return
      }

      respond(false, null, `unsupported method: ${method}`)
    } catch (error) {
      respond(false, null, String(error))
    }
  }

  async function onIframeMessage(event) {
    const msg = event?.data
    if (!msg) return

    const api = appApi()
    if (!api) return

    if (msg?.type === 'talos:bridge:from-iframe') {
      api.IframePostToHost(msg.app_id || '', msg.event || 'message', String(msg.payload || ''))
      return
    }
    if (msg?.type === 'talos:sdk:req') {
      await handleIframeSDKRequest(msg, event.source)
    }
  }

  onMount(async () => {
    await refresh()
    window.addEventListener('message', onIframeMessage)
    EventsOn('packages:event', async (evt) => {
      eventLog = [evt, ...eventLog].slice(0, 12)
      await refresh()
    })
    EventsOn('permissions:request', (evt) => {
      permissionEvents = [evt, ...permissionEvents].slice(0, 8)
      pendingPermissions = [evt, ...pendingPermissions].slice(0, 8)
    })
    EventsOn('iframe:from', (evt) => {
      iframeEvents = [evt, ...iframeEvents].slice(0, 8)
    })
    EventsOn('iframe:to', (evt) => {
      const frame = document.getElementById(`iframe-${evt.app_id}`)
      frame?.contentWindow?.postMessage(
        {
          type: 'talos:bridge:from-host',
          app_id: evt.app_id,
          event: evt.event,
          payload: evt.payload
        },
        '*'
      )
    })
    const ids = Object.keys(packageMap)
    if (ids.length > 0) {
      selectedAppId = ids[0]
    }
  })
</script>

<main>
  <h1>Talos Phase 2 (Started)</h1>
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
            <button on:click={() => grantFsExternal(id)}>Grant fs:external</button>
          </div>
          {#if getIframeSrc(pkg)}
            <iframe id={"iframe-" + id} title={"tiny-app-" + id} src={getIframeSrc(pkg)} on:load={(e) => onIframeLoad(e, id)} />
          {:else}
            <iframe id={"iframe-" + id} title={"tiny-app-" + id} srcdoc="<html><body style='font-family:sans-serif'>Tiny App Shell</body></html>" on:load={(e) => onIframeLoad(e, id)} />
          {/if}
        </li>
      {/each}
    </ul>
  {/if}

  <h2>Iframe Bridge (Host -> Iframe)</h2>
  <div class="controls">
    <input bind:value={selectedAppId} placeholder="app id" />
    <input bind:value={iframeEvent} placeholder="event" />
    <input bind:value={iframePayload} placeholder="payload" />
    <button on:click={sendToIframe}>Send</button>
  </div>

  <h2>Discovery Events</h2>
  <ul>
    {#each eventLog as evt}
      <li><code>{evt.type}</code> {evt.package_id || evt.error}</li>
    {/each}
  </ul>

  <h2>Permission Requests</h2>
  {#if pendingPermissions.length > 0}
    <ul>
      {#each pendingPermissions as evt}
        <li>
          <code>{evt.app_id}</code> requests <code>{evt.scope}</code> ({evt.reason})
          <button on:click={() => decidePermission(evt.app_id, evt.scope, true)}>Allow</button>
          <button on:click={() => decidePermission(evt.app_id, evt.scope, false)}>Deny</button>
        </li>
      {/each}
    </ul>
  {/if}
  <ul>
    {#each permissionEvents as evt}
      <li><code>{evt.app_id}</code> requested <code>{evt.scope}</code> ({evt.reason})</li>
    {/each}
  </ul>

  <h2>Iframe Events</h2>
  <ul>
    {#each iframeEvents as evt}
      <li><code>{evt.app_id}</code> <code>{evt.event}</code> {evt.payload}</li>
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
  .controls {
    display: flex;
    gap: 8px;
    margin-bottom: 8px;
  }
  input {
    min-width: 120px;
  }
  iframe {
    width: 100%;
    height: 64px;
    border: 1px solid #999;
    margin-top: 8px;
    background: #fff;
  }
</style>
