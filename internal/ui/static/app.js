// ==============================================
//  Meridian Admin UI — Alpine.js root component
// ==============================================

function app() {
  return {
    // ── State ──────────────────────────────────
    sources:      [],
    layers:       [],
    styles:       [],
    metrics:      { requests: 0, getmap: 0, cacheHits: 0, errors: 0 },
    health:       'ok',
    activeTab:    'layers',
    form: {
      open: false,
      mode: 'create',   // 'create' | 'edit'
      type: 'layer',    // 'source' | 'layer' | 'style'
      data: {},
    },
    deleteConfirm: null,  // { type, id } | null
    submitting:    false,
    wmsTiles:     {},     // layerID → Leaflet WMS layer object
    map:          null,
    identifyEnabled: true,

    // Logs
    logsLevel:   'ALL',
    logsLimit:   200,
    logsEntries: [],
    logsTotal:   0,
    logsStatus:  '',
    logsTail:    false,
    logsPollId:  null,
    logsLoading: false,

    // ── Init ───────────────────────────────────
    async init() {
      await this.loadAll();
      this.checkHealth();
      this.loadMetrics();
      setInterval(() => this.checkHealth(),  30_000);
      setInterval(() => this.loadMetrics(),  10_000);
      this.$nextTick(() => this.initMap());
    },

    async loadAll() {
      await Promise.all([
        this.loadSources(),
        this.loadLayers(),
        this.loadStyles(),
      ]);
    },

    async loadSources() {
      const res = await fetch('/api/v1/sources');
      if (res.ok) this.sources = await res.json();
    },

    async loadLayers() {
      const res = await fetch('/api/v1/layers');
      if (res.ok) this.layers = await res.json();
    },

    async loadStyles() {
      const res = await fetch('/api/v1/styles');
      if (res.ok) this.styles = await res.json();
    },

    // ── Health ─────────────────────────────────
    async checkHealth() {
      try {
        const res = await fetch('/health');
        this.health = res.ok ? 'ok' : 'error';
      } catch {
        this.health = 'error';
      }
    },

    // ── Metrics ────────────────────────────────
    async loadMetrics() {
      try {
        const res  = await fetch('/metrics');
        const text = await res.text();
        this.metrics = parseMetrics(text);
      } catch { /* keep last values */ }
    },

    // ── Tabs ───────────────────────────────────
    switchTab(tab) {
      if (this.activeTab === 'logs' && tab !== 'logs') {
        this.stopLogsTail();
      }
      this.activeTab = tab;
      this.closeForm();
      this.deleteConfirm = null;
      if (tab === 'logs') this.loadLogs();
    },

    // ── Form helpers ───────────────────────────
    formTitle() {
      const action = this.form.mode === 'edit' ? 'Edit' : 'Add';
      const type   = this.form.type.charAt(0).toUpperCase() + this.form.type.slice(1);
      return `${action} ${type}`;
    },

    openForm(type, mode, data = {}) {
      this.deleteConfirm = null;
      this.form = { open: true, type, mode, data: { ...data } };
    },

    closeForm() {
      if (this.$refs.fileInput) this.$refs.fileInput.value = '';
      this.form = { open: false, mode: 'create', type: this.form.type, data: {} };
    },

    openLayerEditForm(layer) {
      let srs;
      try   { srs = JSON.parse(layer.SRS || '["EPSG:4326"]'); }
      catch { srs = ['EPSG:4326']; }

      this.openForm('layer', 'edit', {
        ID:           layer.ID,
        name:         layer.Name,
        title:        layer.Title,
        source_id:    layer.SourceID,
        source_layer: layer.SourceLayer,
        style_id:     layer.StyleID,
        srs,
        bboxMinLon:   layer.MinX,
        bboxMinLat:   layer.MinY,
        bboxMaxLon:   layer.MaxX,
        bboxMaxLat:   layer.MaxY,
        showOnMap:    this.isLayerVisible(layer.ID),
      });
    },

    toggleSrs(crs) {
      const srs = this.form.data.srs || [];
      const idx = srs.indexOf(crs);
      this.form.data.srs = idx === -1
        ? [...srs, crs]
        : srs.filter(s => s !== crs);
    },

    sourceTypeForId(id) {
      const src = this.sources.find(s => s.ID === id);
      return src ? src.Type : '';
    },

    async submitForm() {
      if (this.submitting) return;
      this.submitting = true;
      try {
        const { type, mode } = this.form;
        if (type === 'source') await this.createSource();
        else if (type === 'layer')  await (mode === 'edit' ? this.updateLayer() : this.createLayer());
        else if (type === 'style')  await this.createStyle();
      } finally {
        this.submitting = false;
      }
    },

    // ── Sources ────────────────────────────────
    async createSource() {
      if (!this.form.data.name) { alert('Name is required'); return; }

      // GeoJSON: upload file first, then create source record
      if (this.form.data.type === 'geojson') {
        if (!this.form.data.file) { alert('Please select a GeoJSON file'); return; }
        let upData;
        try {
          const fd = new FormData();
          fd.append('file', this.form.data.file);
          const up = await fetch('/api/v1/upload', { method: 'POST', body: fd });
          if (!up.ok) { alert(`Upload failed: ${up.status}`); return; }
          upData = await up.json();
        } catch { alert('Upload failed — network error'); return; }

        const geoConfig = upData.config || (upData.path ? { path: upData.path } : null);
        if (!geoConfig) { alert('Upload response missing config'); return; }

        try {
          const res = await fetch('/api/v1/sources', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({
              name:   this.form.data.name,
              type:   'geojson',
              config: geoConfig,
            }),
          });
          if (!res.ok) { alert(`Failed to create source: ${res.status}`); return; }
          await this.loadSources();
          this.closeForm();
        } catch { alert('Network error — could not create source'); }
        return;
      }

      // PostGIS: parse JSON config and create source record
      let config;
      try   { config = JSON.parse(this.form.data.config || '{}'); }
      catch { alert('Config must be valid JSON'); return; }

      try {
        const res = await fetch('/api/v1/sources', {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({
            name:   this.form.data.name,
            type:   this.form.data.type,
            config,
          }),
        });
        if (!res.ok) { alert(`Failed to create source: ${res.status}`); return; }
        await this.loadSources();
        this.closeForm();
      } catch { alert('Network error — could not create source'); }
    },

    async deleteSource(id) {
      const prev = this.sources;
      this.sources      = this.sources.filter(s => s.ID !== id);
      this.deleteConfirm = null;

      try {
        const res = await fetch(`/api/v1/sources/${id}`, { method: 'DELETE' });
        if (!res.ok) this.sources = prev;
      } catch { this.sources = prev; }
    },

    // ── Layers ─────────────────────────────────
    _layerBody() {
      return {
        name:         this.form.data.name,
        title:        this.form.data.title        || '',
        source_id:    this.form.data.source_id    || '',
        source_layer: this.form.data.source_layer || '',
        style_id:     this.form.data.style_id     || '',
        srs:          this.form.data.srs          || ['EPSG:4326'],
        bbox: [
          this.form.data.bboxMinLon ?? -180,
          this.form.data.bboxMinLat ?? -90,
          this.form.data.bboxMaxLon ?? 180,
          this.form.data.bboxMaxLat ?? 90,
        ],
      };
    },

    async createLayer() {
      try {
        const res = await fetch('/api/v1/layers', {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify(this._layerBody()),
        });
        if (!res.ok) { alert(`Failed to create layer: ${res.status}`); return; }
        const layer = await res.json();
        await this.loadLayers();
        if (this.form.data.showOnMap !== false) this.addWmsTile(layer);
        this.closeForm();
      } catch { alert('Network error — could not create layer'); }
    },

    async updateLayer() {
      const id = this.form.data.ID;
      try {
        const res = await fetch(`/api/v1/layers/${id}`, {
          method:  'PUT',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify(this._layerBody()),
        });
        if (!res.ok) { alert(`Failed to update layer: ${res.status}`); return; }
        this.removeWmsTile(id);
        await this.loadLayers();
        const updated = this.layers.find(l => l.ID === id);
        if (updated && this.form.data.showOnMap !== false) this.addWmsTile(updated);
        this.closeForm();
      } catch { alert('Network error — could not update layer'); }
    },

    async deleteLayer(id) {
      const prev       = this.layers;
      const wasVisible = this.isLayerVisible(id);
      this.layers        = this.layers.filter(l => l.ID !== id);
      this.deleteConfirm = null;
      this.removeWmsTile(id);

      try {
        const res = await fetch(`/api/v1/layers/${id}`, { method: 'DELETE' });
        if (!res.ok) {
          this.layers = prev;
          if (wasVisible) {
            const layer = prev.find(l => l.ID === id);
            if (layer) this.addWmsTile(layer);
          }
        }
      } catch {
        this.layers = prev;
        if (wasVisible) {
          const layer = prev.find(l => l.ID === id);
          if (layer) this.addWmsTile(layer);
        }
      }
    },

    // ── Styles ─────────────────────────────────
    async createStyle() {
      try {
        const res = await fetch('/api/v1/styles', {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({
            name:         this.form.data.name,
            fill_color:   this.form.data.fill_color   || '#3388ff',
            stroke_color: this.form.data.stroke_color || '#ffffff',
            stroke_width: this.form.data.stroke_width ?? 1.0,
            opacity:      this.form.data.opacity      ?? 1.0,
          }),
        });
        if (!res.ok) { alert(`Failed to create style: ${res.status}`); return; }
        await this.loadStyles();
        this.closeForm();
      } catch { alert('Network error — could not create style'); }
    },

    async deleteStyle(id) {
      const prev = this.styles;
      this.styles        = this.styles.filter(s => s.ID !== id);
      this.deleteConfirm = null;

      try {
        const res = await fetch(`/api/v1/styles/${id}`, { method: 'DELETE' });
        if (!res.ok) this.styles = prev;
      } catch { this.styles = prev; }
    },

    // ── Map ────────────────────────────────────
    initMap() {
      if (!window.L) return;
      this.map = L.map('map').setView([26.5, 30.8], 5);
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© <a href="https://openstreetmap.org/copyright">OpenStreetMap</a> contributors',
        maxZoom: 19,
      }).addTo(this.map);
      // Add WMS tiles for all existing layers on load
      this.layers.forEach(l => this.addWmsTile(l));
      // Keep first-time UX deterministic: focus the built-in test layer if present.
      const testLayer = this.layers.find(l => l.Name === 'test_layer');
      if (testLayer) this.zoomToLayer(testLayer);
      this.map.on('click', (e) => this.identifyAt(e));
    },

    addWmsTile(layer) {
      if (!this.map) return;
      // Remove any stale tile for this layer first
      if (this.wmsTiles[layer.ID]) {
        this.map.removeLayer(this.wmsTiles[layer.ID]);
      }
      const tile = L.tileLayer.wms('/wms', {
        layers:      layer.Name,
        format:      'image/png',
        transparent: true,
        version:     '1.3.0',
        maxZoom:     22,
        // No crs override — inherit the map's default EPSG:3857 so WMS
        // requests and tile projection stay consistent.
      });
      tile.addTo(this.map);
      this.wmsTiles[layer.ID] = tile;
      this.zoomToLayer(layer);
    },

    zoomToLayer(layer) {
      if (!this.map) return;
      const minLon = layer.MinX, minLat = layer.MinY;
      const maxLon = layer.MaxX, maxLat = layer.MaxY;
      if (minLon === maxLon || minLat === maxLat) return; // degenerate bbox
      this.map.fitBounds([[minLat, minLon], [maxLat, maxLon]], { padding: [20, 20] });
    },

    removeWmsTile(layerID) {
      if (this.wmsTiles[layerID] && this.map) {
        this.map.removeLayer(this.wmsTiles[layerID]);
        delete this.wmsTiles[layerID];
      }
    },

    toggleWmsTile(layer) {
      if (this.wmsTiles[layer.ID]) {
        this.removeWmsTile(layer.ID);
      } else {
        this.addWmsTile(layer);
      }
    },

    isLayerVisible(layerID) {
      return !!this.wmsTiles[layerID];
    },

    topVisibleLayer() {
      for (let i = this.layers.length - 1; i >= 0; i--) {
        const layer = this.layers[i];
        if (this.isLayerVisible(layer.ID)) return layer;
      }
      return null;
    },

    async identifyAt(e) {
      if (!this.identifyEnabled || !this.map) return;
      const layer = this.topVisibleLayer();
      if (!layer) return;

      const size = this.map.getSize();
      const bounds = this.map.getBounds();
      const sw = bounds.getSouthWest();
      const ne = bounds.getNorthEast();
      const p = this.map.latLngToContainerPoint(e.latlng);

      const params = new URLSearchParams({
        SERVICE: 'WMS',
        VERSION: '1.3.0',
        REQUEST: 'GetFeatureInfo',
        LAYERS: layer.Name,
        QUERY_LAYERS: layer.Name,
        STYLES: '',
        CRS: 'EPSG:4326',
        BBOX: `${sw.lat},${sw.lng},${ne.lat},${ne.lng}`, // WMS 1.3.0 axis order for EPSG:4326
        WIDTH: String(Math.round(size.x)),
        HEIGHT: String(Math.round(size.y)),
        I: String(Math.round(p.x)),
        J: String(Math.round(p.y)),
        INFO_FORMAT: 'application/json',
        FEATURE_COUNT: '10',
      });

      try {
        const res = await fetch(`/wms?${params.toString()}`);
        if (!res.ok) return;
        const data = await res.json();
        const features = Array.isArray(data.features) ? data.features : [];
        if (features.length === 0) {
          L.popup().setLatLng(e.latlng).setContent('No feature found').openOn(this.map);
          return;
        }
        L.popup()
          .setLatLng(e.latlng)
          .setContent(this.featureInfoHTML(layer.Name, features))
          .openOn(this.map);
      } catch {
        // Ignore identify network errors to keep map interaction fluid.
      }
    },

    featureInfoHTML(layerName, features) {
      const rows = [];
      for (const f of features) {
        const props = f.properties || {};
        const keys = Object.keys(props);
        if (keys.length === 0) {
          rows.push('<div><em>No properties</em></div>');
          continue;
        }
        for (const k of keys) {
          rows.push(
            `<div><strong>${escapeHTML(k)}:</strong> ${escapeHTML(String(props[k]))}</div>`
          );
        }
      }
      return `<div style="min-width:220px"><div><strong>${escapeHTML(layerName)}</strong></div>${rows.join('')}</div>`;
    },

    stopLogsTail() {
      if (this.logsPollId) {
        clearInterval(this.logsPollId);
        this.logsPollId = null;
      }
    },

    async toggleLogsTail() {
      if (this.logsTail) {
        this.stopLogsTail();
        await this.loadLogs();
        this.logsPollId = setInterval(() => this.loadLogs(), 5000);
      } else {
        this.stopLogsTail();
      }
    },

    async loadLogs() {
      if (this.logsLoading) return;
      this.logsLoading = true;
      this.logsStatus = 'Loading…';
      const level  = this.logsLevel === 'ALL' ? '' : this.logsLevel;
      const params = new URLSearchParams({ limit: this.logsLimit });
      if (level) params.set('level', level);
      try {
        const res = await fetch(`/api/v1/logs?${params}`);
        if (!res.ok) { this.logsStatus = `Error ${res.status}`; return; }
        const data = await res.json();
        this.logsEntries = (data.entries || []).reverse(); // newest first
        this.logsTotal   = data.total || 0;
        this.logsStatus  = `${this.logsEntries.length} entries (${this.logsTotal} total)`;
      } catch {
        this.logsStatus = 'Network error';
      } finally {
        this.logsLoading = false;
      }
    },

    levelClass(level) {
      if (level === 'ERROR') return 'log-error';
      if (level === 'WARN')  return 'log-warn';
      if (level === 'DEBUG') return 'log-debug';
      return 'log-info';
    },

    formatLogTime(timeStr) {
      if (!timeStr) return '';
      try {
        return new Date(timeStr).toLocaleTimeString();
      } catch { return timeStr; }
    },
  };
}

// ── Metrics parser ─────────────────────────────
function parseMetrics(text) {
  const get = (name) => {
    const m = text.match(new RegExp(`^${name}\\s+(\\d+)`, 'm'));
    return m ? parseInt(m[1], 10) : 0;
  };
  return {
    requests:  get('meridian_wms_requests_total'),
    getmap:    get('meridian_wms_getmap_total'),
    cacheHits: get('meridian_cache_hits_total'),
    errors:    get('meridian_wms_errors_total'),
  };
}

function escapeHTML(v) {
  return v
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}
