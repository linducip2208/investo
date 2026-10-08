class ChartDrawings {
  constructor() {
    this.drawings = [];
    this.undoStack = [];
    this.redoStack = [];
    this.maxUndo = 50;
    this.activeTool = null;
    this.seriesMap = {};
    this.nextId = 1;
    this.listeners = [];
  }

  onChange(fn) { this.listeners.push(fn); }
  _notify() { this.listeners.forEach(fn => fn(this.drawings)); }

  _generateId() { return 'dwg_' + (this.nextId++); }

  setActiveTool(tool) { this.activeTool = tool; }

  getDrawings() { return this.drawings; }

  addDrawing(drawing) {
    this.undoStack.push(JSON.parse(JSON.stringify(this.drawings)));
    if (this.undoStack.length > this.maxUndo) this.undoStack.shift();
    this.redoStack = [];
    drawing.id = drawing.id || this._generateId();
    this.drawings.push(drawing);
    this._notify();
    return drawing;
  }

  removeDrawing(id) {
    this.undoStack.push(JSON.parse(JSON.stringify(this.drawings)));
    if (this.undoStack.length > this.maxUndo) this.undoStack.shift();
    this.redoStack = [];
    this.drawings = this.drawings.filter(d => d.id !== id);
    this._notify();
  }

  updateDrawing(id, updates) {
    this.undoStack.push(JSON.parse(JSON.stringify(this.drawings)));
    if (this.undoStack.length > this.maxUndo) this.undoStack.shift();
    this.redoStack = [];
    const idx = this.drawings.findIndex(d => d.id === id);
    if (idx >= 0) {
      this.drawings[idx] = { ...this.drawings[idx], ...updates };
    }
    this._notify();
  }

  undo() {
    if (this.undoStack.length === 0) return false;
    this.redoStack.push(JSON.parse(JSON.stringify(this.drawings)));
    if (this.redoStack.length > this.maxUndo) this.redoStack.shift();
    this.drawings = this.undoStack.pop();
    this._notify();
    return true;
  }

  redo() {
    if (this.redoStack.length === 0) return false;
    this.undoStack.push(JSON.parse(JSON.stringify(this.drawings)));
    if (this.undoStack.length > this.maxUndo) this.undoStack.shift();
    this.drawings = this.redoStack.pop();
    this._notify();
    return true;
  }

  clearAll() {
    this.undoStack.push(JSON.parse(JSON.stringify(this.drawings)));
    if (this.undoStack.length > this.maxUndo) this.undoStack.shift();
    this.redoStack = [];
    this.drawings = [];
    this._notify();
  }

  saveLocalStorage(key) {
    try {
      localStorage.setItem(key, JSON.stringify(this.drawings));
      return true;
    } catch (e) { return false; }
  }

  loadLocalStorage(key) {
    try {
      const data = localStorage.getItem(key);
      if (data) {
        this.undoStack.push(JSON.parse(JSON.stringify(this.drawings)));
        this.redoStack = [];
        this.drawings = JSON.parse(data);
        this._notify();
        return true;
      }
    } catch (e) { return false; }
    return false;
  }

  _renderOnChart(mainChart, mainCandle, drawingsToRender, visibleRange, colors) {
    drawingsToRender.forEach(d => {
      if (d.type === 'trendline') {
        this._renderTrendline(mainChart, d, visibleRange, colors);
      } else if (d.type === 'horizontal_line') {
        this._renderHorizontalLine(mainChart, d, visibleRange, colors);
      } else if (d.type === 'fib_retracement') {
        this._renderFibRetracement(mainChart, d, visibleRange, colors);
      } else if (d.type === 'rectangle') {
        this._renderRectangle(mainChart, d, visibleRange, colors);
      } else if (d.type === 'text_label') {
        this._renderTextLabel(mainChart, d, colors);
      }
    });
  }

  _renderTrendline(mainChart, d, visibleRange, colors) {
    if (!d.points || d.points.length < 2) return;
    const line = mainChart.addLineSeries({
      color: d.color || '#3b82f6',
      lineWidth: d.lineWidth || 2,
      lineStyle: d.lineStyle === 'dashed' ? LightweightCharts.LineStyle.Dashed : LightweightCharts.LineStyle.Solid,
      priceLineVisible: false,
      lastValueVisible: false,
    });
    const data = d.points.map(p => ({ time: p.time, value: p.price }));
    line.setData(data);
    if (!this.seriesMap[d.id]) this.seriesMap[d.id] = [];
    this.seriesMap[d.id].push(line);
  }

  _renderHorizontalLine(mainChart, d, visibleRange, colors) {
    if (!visibleRange.from || !visibleRange.to) return;
    const line = mainChart.addLineSeries({
      color: d.color || '#f59e0b',
      lineWidth: d.lineWidth || 1.5,
      lineStyle: LightweightCharts.LineStyle.Dashed,
      priceLineVisible: false,
      lastValueVisible: true,
    });
    line.setData([
      { time: visibleRange.from, value: d.price },
      { time: visibleRange.to, value: d.price },
    ]);
    if (!this.seriesMap[d.id]) this.seriesMap[d.id] = [];
    this.seriesMap[d.id].push(line);
  }

  _renderFibRetracement(mainChart, d, visibleRange, colors) {
    if (!d.high || !d.low || !visibleRange.from || !visibleRange.to) return;
    const diff = d.high - d.low;
    if (diff <= 0) return;
    const levels = [
      { label: '0%', ratio: 0, color: 'rgba(16,185,129,0.6)' },
      { label: '23.6%', ratio: 0.236, color: 'rgba(59,130,246,0.5)' },
      { label: '38.2%', ratio: 0.382, color: 'rgba(99,102,241,0.5)' },
      { label: '50%', ratio: 0.5, color: 'rgba(245,158,11,0.5)' },
      { label: '61.8%', ratio: 0.618, color: 'rgba(245,158,11,0.6)' },
      { label: '78.6%', ratio: 0.786, color: 'rgba(239,68,68,0.5)' },
      { label: '100%', ratio: 1, color: 'rgba(239,68,68,0.6)' },
    ];
    levels.forEach(l => {
      const price = d.high - diff * l.ratio;
      const line = mainChart.addLineSeries({
        color: l.color,
        lineWidth: 1,
        lineStyle: LightweightCharts.LineStyle.Dashed,
        priceLineVisible: false,
        lastValueVisible: true,
      });
      line.setData([
        { time: visibleRange.from, value: Math.round(price * 100) / 100 },
        { time: visibleRange.to, value: Math.round(price * 100) / 100 },
      ]);
      if (!this.seriesMap[d.id]) this.seriesMap[d.id] = [];
      this.seriesMap[d.id].push(line);
    });
  }

  _renderRectangle(mainChart, d, visibleRange, colors) {
    if (!d.bounds || !visibleRange.from || !visibleRange.to) return;
    const { topLeft, bottomRight } = d.bounds;
    if (!topLeft || !bottomRight) return;

    const fillColor = d.fillColor || 'rgba(59,130,246,0.15)';
    const borderColor = d.color || '#3b82f6';

    const topLine = mainChart.addLineSeries({
      color: borderColor,
      lineWidth: 1,
      priceLineVisible: false,
      lastValueVisible: false,
    });
    topLine.setData([
      { time: topLeft.time, value: topLeft.price },
      { time: bottomRight.time, value: topLeft.price },
    ]);

    const bottomLine = mainChart.addLineSeries({
      color: borderColor,
      lineWidth: 1,
      priceLineVisible: false,
      lastValueVisible: false,
    });
    bottomLine.setData([
      { time: topLeft.time, value: bottomRight.price },
      { time: bottomRight.time, value: bottomRight.price },
    ]);

    const leftLine = mainChart.addLineSeries({
      color: borderColor,
      lineWidth: 1,
      priceLineVisible: false,
      lastValueVisible: false,
    });
    leftLine.setData([
      { time: topLeft.time, value: bottomRight.price },
      { time: topLeft.time, value: topLeft.price },
    ]);

    const rightLine = mainChart.addLineSeries({
      color: borderColor,
      lineWidth: 1,
      priceLineVisible: false,
      lastValueVisible: false,
    });
    rightLine.setData([
      { time: bottomRight.time, value: bottomRight.price },
      { time: bottomRight.time, value: topLeft.price },
    ]);

    if (!this.seriesMap[d.id]) this.seriesMap[d.id] = [];
    this.seriesMap[d.id].push(topLine, bottomLine, leftLine, rightLine);
  }

  _renderTextLabel(mainChart, d, colors) {
    if (d.position) {
      const markerColor = d.color || '#6b7280';
      if (mainChart._activeCandleSeries) {
        mainChart._activeCandleSeries.setMarkers(mainChart._activeCandleSeries.markers ? [
          ...mainChart._activeCandleSeries.markers(),
          {
            time: d.position.time,
            position: 'aboveBar',
            color: markerColor,
            shape: 'circle',
            text: d.text || 'Label',
            size: 2,
          }
        ] : []);
      }
    }
  }

  clearSeriesFromChart(mainChart) {
    Object.values(this.seriesMap).forEach(seriesArr => {
      seriesArr.forEach(s => {
        try { mainChart.removeSeries(s); } catch (e) { }
      });
    });
    this.seriesMap = {};
  }

  renderAll(mainChart, mainCandle, visibleRange, colors) {
    this.clearSeriesFromChart(mainChart);
    mainChart._activeCandleSeries = mainCandle;
    this._renderOnChart(mainChart, mainCandle, this.drawings, visibleRange, colors);
  }
}

window.ChartDrawings = ChartDrawings;
