const pptxgen = require('pptxgenjs');
const {
  imageSizingCrop,
  imageSizingContain,
  safeOuterShadow,
  warnIfSlideHasOverlaps,
  warnIfSlideElementsOutOfBounds,
} = require('./pptxgenjs_helpers');

const pptx = new pptxgen();
pptx.layout = 'LAYOUT_WIDE';
pptx.author = 'AIoT Care Sensing Team';
pptx.subject = '無接觸 AIoT 照護感測產品商業簡報';
pptx.title = 'CARE SENSE｜無接觸 AIoT 照護感測系統';
pptx.company = 'CARE SENSE';
pptx.lang = 'zh-TW';
pptx.theme = {
  headFontFace: 'PingFang TC',
  bodyFontFace: 'PingFang TC',
  lang: 'zh-TW',
};
pptx.defineSlideMaster({
  title: 'MASTER',
  background: { color: 'FBF2E3' },
  objects: [
    { rect: { x: 0, y: 0, w: 13.333, h: 0.08, fill: { color: 'FF4048' }, line: { color: 'FF4048' } } },
    { text: { text: 'CARE SENSE  ·  CONTACTLESS CARE INTELLIGENCE', options: { x: 0.55, y: 7.13, w: 6.8, h: 0.18, fontFace: 'PingFang TC', fontSize: 6.5, color: '866F5D', charSpacing: 1.4, margin: 0 } } },
  ],
  slideNumber: { x: 12.35, y: 7.08, w: 0.45, h: 0.22, color: '866F5D', fontFace: 'PingFang TC', fontSize: 7, align: 'right', margin: 0 },
});

const C = {
  bg: 'FBF2E3', panel: 'F4E3C9', panel2: 'E9D7B9', ink: '4A2C21', muted: '866F5D',
  cyan: '8D9C78', cyan2: '66775A', blue: '7D9897', orange: 'E1A55B',
  magenta: 'D98F7A', green: '7D9369', red: 'FF4048', line: 'D8C1A3', white: 'FFFFFF',
};
const A = {
  productFront: 'assets/product-front-flat.png',
  productSide: 'assets/product-side.png',
  productAngle: 'assets/product-front-angle.png',
  commandCenter: 'assets/command-center-ui.png',
};
const shadow = safeOuterShadow('000000', 0.28, 45, 2, 1);

function addSlide() { return pptx.addSlide('MASTER'); }
function rect(slide, x, y, w, h, fill, radius = 0.12, line = fill) {
  slide.addShape(pptx.ShapeType.roundRect, { x, y, w, h, rectRadius: radius, fill: { color: fill }, line: { color: line, transparency: line === fill ? 100 : 0, width: 1 } });
}
function line(slide, x, y, w, h, color = C.line, width = 1, dash = 'solid') {
  slide.addShape(pptx.ShapeType.line, { x, y, w, h, line: { color, width, dashType: dash, beginArrowType: 'none', endArrowType: 'none' } });
}
function text(slide, value, x, y, w, h, size = 18, color = C.ink, opts = {}) {
  slide.addText(value, { x, y, w, h, fontFace: 'PingFang TC', fontSize: size, color, margin: 0, breakLine: false, valign: 'mid', ...opts });
}
function title(slide, kicker, heading, sub = '') {
  text(slide, kicker.toUpperCase(), 0.58, 0.34, 4.8, 0.24, 7.5, C.cyan, { bold: true, charSpacing: 1.8 });
  text(slide, heading, 0.58, 0.66, 12.0, 0.6, 26, C.ink, { bold: true });
  if (sub) text(slide, sub, 0.6, 1.28, 11.6, 0.34, 10.5, C.muted);
}
function pill(slide, label, x, y, w, color = C.cyan, fill = C.panel2) {
  rect(slide, x, y, w, 0.34, fill, 0.16, color);
  text(slide, label, x, y + 0.01, w, 0.3, 8.5, color, { bold: true, align: 'center' });
}
function metricCard(slide, x, y, w, h, number, label, accent = C.cyan, note = '') {
  rect(slide, x, y, w, h, C.panel, 0.14, C.line);
  slide.addShape(pptx.ShapeType.rect, { x, y, w: 0.07, h, fill: { color: accent }, line: { color: accent } });
  text(slide, number, x + 0.24, y + 0.18, w - 0.45, 0.42, 23, C.ink, { bold: true });
  text(slide, label, x + 0.24, y + 0.66, w - 0.45, 0.28, 9.5, C.muted, { bold: true });
  if (note) text(slide, note, x + 0.24, y + 1.0, w - 0.45, h - 1.12, 8, C.muted, { valign: 'top', breakLine: true });
}
function dot(slide, x, y, color, r = 0.08) {
  slide.addShape(pptx.ShapeType.ellipse, { x, y, w: r, h: r, fill: { color }, line: { color } });
}
function footerNote(slide, value) { text(slide, value, 0.6, 6.78, 11.8, 0.22, 6.8, C.muted, { italic: true }); }

function addProductImage(slide, path, x, y, w, h) {
  slide.addImage({ path, ...imageSizingContain(path, x, y, w, h) });
}

function addInterfaceScreenshot(slide, path, x, y, w, h) {
  // The screenshot intentionally sits inside a thin presentation frame.
  rect(slide, x - 0.04, y - 0.04, w + 0.08, h + 0.08, 'FFF9F0', 0.14, '9A7F69');
  slide.addImage({ path, ...imageSizingCrop(path, x, y, w, h) });
}

function addSensorField(slide, x, y, w, h) {
  rect(slide, x, y, w, h, 'F4E3C9', 0.18, C.line);
  for (let row = 0; row < 7; row++) {
    for (let col = 0; col < 9; col++) {
      const color = (row + col) % 5 === 0 ? C.cyan : ((row * 3 + col) % 7 === 0 ? C.blue : C.line);
      dot(slide, x + 0.42 + col * ((w - 0.84) / 8), y + 0.42 + row * ((h - 1.05) / 6), color, 0.07);
    }
  }
  line(slide, x + 0.38, y + h - 0.45, w - 0.76, 0, C.cyan, 1.2);
  text(slide, 'CONTACTLESS SENSOR FIELD', x + 0.4, y + h - 0.34, w - 0.8, 0.2, 6.5, C.muted, { charSpacing: 1.3 });
}

function addCareEventCard(slide, x, y, w, h, number, label, accent, kind) {
  rect(slide, x, y, w, h, C.panel, 0.16, C.line);
  text(slide, number, x + 0.25, y + 0.22, 0.45, 0.26, 9, accent, { bold: true });
  const cx = x + w / 2;
  if (kind === 'presence') {
    slide.addShape(pptx.ShapeType.ellipse, { x: cx - 0.34, y: y + 0.72, w: 0.68, h: 0.68, fill: { color: accent, transparency: 68 }, line: { color: accent, width: 1.5 } });
    rect(slide, cx - 0.62, y + 1.46, 1.24, 0.95, 'E9D7B9', 0.25, accent);
    line(slide, x + 0.55, y + 2.58, w - 1.1, 0, accent, 2);
  } else if (kind === 'care') {
    slide.addShape(pptx.ShapeType.arc, { x: cx - 0.82, y: y + 0.7, w: 1.65, h: 1.65, adjustPoint: 0.3, rotate: 18, fill: { color: C.bg, transparency: 100 }, line: { color: accent, width: 2.2, beginArrowType: 'none', endArrowType: 'triangle' } });
    slide.addShape(pptx.ShapeType.ellipse, { x: cx - 0.28, y: y + 1.15, w: 0.56, h: 0.56, fill: { color: accent, transparency: 55 }, line: { color: accent } });
    line(slide, cx - 0.7, y + 2.38, 1.4, 0, C.line, 4);
  } else {
    rect(slide, x + 0.55, y + 1.0, w - 1.1, 1.2, 'E9D7B9', 0.08, C.line);
    line(slide, x + 0.78, y + 0.75, 0, 1.72, accent, 2, 'dash');
    line(slide, x + w - 0.78, y + 0.75, 0, 1.72, accent, 2, 'dash');
    slide.addShape(pptx.ShapeType.ellipse, { x: cx - 0.23, y: y + 1.25, w: 0.46, h: 0.46, fill: { color: accent }, line: { color: accent } });
  }
  text(slide, label, x + 0.3, y + h - 0.58, w - 0.6, 0.3, 12, C.ink, { bold: true, align: 'center' });
}

function addDashboardWireframe(slide, x, y, w, h) {
  rect(slide, x, y, w, h, 'FFF9F0', 0.14, '9A7F69');
  rect(slide, x + 0.18, y + 0.18, w - 0.36, 0.42, '5A3A2C', 0.08, '5A3A2C');
  text(slide, 'AIoT CARE COMMAND CENTER', x + 0.38, y + 0.25, 3.4, 0.18, 7, 'FFF3E4', { bold: true, charSpacing: 1 });
  for (let row = 0; row < 3; row++) {
    for (let col = 0; col < 4; col++) {
      const bx = x + 0.3 + col * 1.18;
      const by = y + 0.82 + row * 0.82;
      rect(slide, bx, by, 0.98, 0.62, row === 0 && col === 2 ? 'FFFFFF' : 'EADDC8', 0.08, row === 0 && col === 2 ? C.red : 'A98E74');
      text(slide, `020${col + 1}_0${row + 1}`, bx + 0.08, by + 0.08, 0.72, 0.16, 5.8, '4A2C21', { bold: true });
      if (row === 0 && col === 2) {
        dot(slide, bx + 0.18, by + 0.4, C.green, 0.08);
        dot(slide, bx + 0.42, by + 0.4, C.green, 0.08);
        dot(slide, bx + 0.66, by + 0.4, C.orange, 0.08);
      }
    }
  }
  rect(slide, x + 5.15, y + 0.82, w - 5.45, 2.26, 'F4E3C9', 0.1, C.line);
  text(slide, '即時趨勢', x + 5.42, y + 1.02, 1.1, 0.22, 8, C.cyan, { bold: true });
  const pts = [[0,0.75],[0.45,0.48],[0.9,0.62],[1.35,0.22],[1.8,0.5],[2.25,0.3]];
  for (let i = 0; i < pts.length - 1; i++) {
    line(slide, x + 5.4 + pts[i][0], y + 1.6 + pts[i][1], pts[i + 1][0] - pts[i][0], pts[i + 1][1] - pts[i][1], C.cyan, 2);
  }
  rect(slide, x + 0.3, y + 3.48, w - 0.6, h - 3.78, 'F0E5D3', 0.08, 'B79C80');
  text(slide, '事件時間軸', x + 0.55, y + 3.7, 1.2, 0.2, 7.5, '4A2C21', { bold: true });
  line(slide, x + 1.85, y + 3.8, w - 2.55, 0, 'A98E74', 1.2);
  [C.green, C.orange, C.blue].forEach((c, i) => dot(slide, x + 3.0 + i * 1.45, y + 3.72, c, 0.12));
}

// 1 — Cover
{
  const s = addSlide();
  s.background = { color: C.bg };
  text(s, 'CARE', 9.02, 0.62, 3.6, 0.8, 38, 'E8D5B7', { fontFace: 'Avenir Next', bold: true, align: 'right', charSpacing: 2.2 });
  text(s, 'SENSE', 9.02, 1.52, 3.6, 0.8, 38, 'E8D5B7', { fontFace: 'Avenir Next', bold: true, align: 'right', charSpacing: 2.2 });
  pill(s, 'AIoT · 無接觸 · 即時', 0.68, 0.62, 2.45, C.cyan2, 'E9D7B9');
  text(s, '把照護量測，', 0.68, 1.55, 7.0, 0.72, 37, C.ink, { bold: true, valign: 'mid' });
  text(s, '從逐床作業\n變成即時感知', 0.68, 2.32, 8.0, 1.52, 43, C.ink, { bold: true, breakLine: true, valign: 'top' });
  line(s, 0.7, 4.2, 11.85, 0, C.red, 2.2);
  text(s, '無接觸 AIoT 照護感測系統', 0.7, 4.48, 5.7, 0.42, 17, C.red, { bold: true, charSpacing: 0.8 });
  text(s, '連續掌握生命表徵、照護動作與床邊安全，\n把資料直接送到需要它的平台。', 0.7, 5.02, 5.9, 0.68, 13, C.muted, { breakLine: true, valign: 'top' });
  const coverValues = [
    ['01', '分擔量測勞力'],
    ['02', '無接觸持續觀察'],
    ['03', '即時警報與串接'],
  ];
  coverValues.forEach((v, i) => {
    const x = 7.18 + i * 1.82;
    rect(s, x, 4.72, 1.6, 1.22, i === 1 ? 'F0DFC4' : 'F7EAD6', 0.14, i === 1 ? C.red : C.line);
    text(s, v[0], x + 0.18, 4.9, 0.42, 0.25, 8.5, i === 1 ? C.red : C.green, { bold: true });
    text(s, v[1], x + 0.18, 5.3, 1.22, 0.42, 9.5, C.ink, { bold: true, breakLine: true, valign: 'top' });
  });
  text(s, 'PRODUCT OVERVIEW  /  2026', 0.7, 6.35, 4.4, 0.25, 8, C.muted, { charSpacing: 1.5 });
}

// 2 — Product prototype
{
  const s = addSlide(); title(s, 'Product prototype', '雙感測模組，讓同一台裝置同時「看見」與「量測」', '上方紅外線熱成像建立溫度分布；下方毫米波捕捉人體微動與生命表徵。');
  rect(s, 0.62, 1.78, 5.0, 4.95, 'F4E3C9', 0.18, C.line);
  addProductImage(s, A.productFront, 0.88, 2.02, 4.48, 3.72);
  // These two labels intentionally overlay the product image to identify the physical modules.
  pill(s, '上方 · 紅外線熱成像', 1.0, 2.08, 2.25, C.red, 'FFF3E4');
  pill(s, '下方 · 毫米波模組', 2.42, 4.92, 2.25, C.green, 'FFF3E4');
  rect(s, 1.02, 5.88, 4.2, 0.54, 'FFF7EA', 0.12, C.line);
  text(s, '雙感測融合：溫度矩陣 ＋ 微動生命表徵', 1.22, 6.02, 3.8, 0.23, 9.5, C.ink, { bold: true, align: 'center' });

  rect(s, 5.92, 1.78, 6.78, 2.25, 'F7E6D2', 0.18, C.red);
  text(s, '01', 6.22, 2.04, 0.45, 0.25, 9, C.red, { bold: true });
  text(s, '紅外線熱成像陣列', 6.78, 1.98, 2.7, 0.38, 17, C.ink, { bold: true });
  text(s, '32 × 24 ＝ 768 個溫度點位', 9.48, 1.99, 2.82, 0.34, 13, C.red, { bold: true, align: 'right' });
  text(s, '每個點位量測物體表面的紅外線輻射並換算溫度，組成 32 × 24 溫度矩陣；系統再透過插值呈現連續分布，供體表溫度、人體輪廓、人在床與床緣事件判定使用。', 6.24, 2.48, 6.0, 0.78, 10.4, C.ink, { breakLine: true, valign: 'top' });
  const thermalTags = ['體表溫度', '人體輪廓', '床緣判定', '低解析隱私'];
  thermalTags.forEach((v, i) => pill(s, v, 6.25 + i * 1.48, 3.42, 1.28, i === 0 ? C.red : C.orange, 'FFF3E4'));

  rect(s, 5.92, 4.28, 6.78, 2.45, 'E9E3D1', 0.18, C.green);
  text(s, '02', 6.22, 4.55, 0.45, 0.25, 9, C.green, { bold: true });
  text(s, '毫米波微動感測模組', 6.78, 4.48, 3.2, 0.38, 17, C.ink, { bold: true });
  text(s, '非接觸取得呼吸、心率與人體微動', 10.05, 4.5, 2.25, 0.34, 10.2, C.green, { bold: true, align: 'right' });
  text(s, '毫米波向人體發射低功率電磁波並接收反射訊號，從胸腔起伏與細微位移中分析週期變化；不需配戴貼片或接觸皮膚，適合長時間觀察呼吸率、心率、存在狀態與動作變化。', 6.24, 5.0, 6.0, 0.78, 10.4, C.ink, { breakLine: true, valign: 'top' });
  const radarTags = ['呼吸率', '心率／心跳', '人體存在', '微動趨勢'];
  radarTags.forEach((v, i) => pill(s, v, 6.25 + i * 1.48, 6.0, 1.28, i === 1 ? C.red : C.green, 'FFF3E4'));
}

// 2 — Promise
{
  const s = addSlide(); title(s, 'Product promise', '一套系統，接手日常量測與持續觀察', '讓照護人員不必一直「去量」，而是在需要介入時收到清楚訊號。');
  metricCard(s, 0.62, 1.9, 2.82, 1.7, '無接觸', '降低配戴與操作負擔', C.cyan, '不需貼片、不需綁帶；以非侵入式感測持續取得生命表徵。');
  metricCard(s, 3.62, 1.9, 2.82, 1.7, '24 / 7', '連續監測與留痕', C.blue, '即時資料、事件與歷史趨勢同步保存，支援追溯與交班。');
  metricCard(s, 6.62, 1.9, 2.82, 1.7, 'AIoT', '邊緣感測＋雲端串接', C.magenta, '從床邊感測、Redis 即時資料到 API 與報表輸出。');
  metricCard(s, 9.62, 1.9, 2.82, 1.7, '一目瞭然', '從數據到可行動警報', C.orange, '把異常、離床、翻身拍背與床緣事件轉成清楚狀態。');
  rect(s, 0.62, 4.02, 11.82, 1.98, C.panel, 0.14, C.line);
  text(s, '核心價值', 0.9, 4.28, 1.3, 0.3, 9, C.cyan, { bold: true });
  text(s, '把照護團隊的時間，從重複量測與抄錄，移回真正需要人的判斷與陪伴。', 0.9, 4.75, 10.9, 0.58, 22, C.ink, { bold: true });
  text(s, '系統是照護決策輔助工具；實際醫療判斷仍由專業人員負責。', 0.92, 5.54, 9.8, 0.23, 7.5, C.muted);
}

// 3 — Pain
{
  const s = addSlide(); title(s, 'Why now', '照護現場的挑戰，不只是人力不足', '量測、巡視、抄錄與回報分散在不同時間點，真正的風險可能出現在兩次巡視之間。');
  const items = [
    ['01', '重複量測', '心率、呼吸與溫度需要逐床操作，時間被例行工作切碎。'],
    ['02', '非連續資訊', '單點數值難以看出變化趨勢，也容易錯過短暫異常。'],
    ['03', '照護事件難留痕', '翻身、拍背與床緣風險往往依賴人工記錄與回憶。'],
    ['04', '資料孤島', '儀器、護理站、報表與院內平台之間缺乏一致介面。'],
  ];
  items.forEach((it, i) => {
    const y = 1.9 + i * 1.12;
    text(s, it[0], 0.72, y, 0.62, 0.42, 18, i === 2 ? C.orange : C.cyan, { bold: true });
    text(s, it[1], 1.52, y, 2.25, 0.36, 14, C.ink, { bold: true });
    text(s, it[2], 3.75, y, 5.0, 0.52, 10.5, C.muted, { valign: 'top' });
    line(s, 1.5, y + 0.72, 7.18, 0, C.line, 0.8);
  });
  text(s, '系統如何回應', 9.28, 1.65, 2.85, 0.22, 9, C.red, { bold: true, charSpacing: 1.2 });
  const responses = [
    ['自動量測', '減少逐床操作'],
    ['趨勢追蹤', '看見連續變化'],
    ['事件辨識', '照護動作自動留痕'],
    ['開放串接', 'API 與報表整合'],
  ];
  responses.forEach((r, i) => {
    const y = 1.9 + i * 1.12;
    rect(s, 9.25, y, 3.1, 0.82, i === 2 ? 'F5DFC8' : C.panel2, 0.12, i === 2 ? C.orange : C.line);
    text(s, String(i + 1).padStart(2, '0'), 9.48, y + 0.17, 0.38, 0.24, 8.5, i === 2 ? C.orange : C.green, { bold: true });
    text(s, r[0], 9.98, y + 0.11, 1.85, 0.25, 11.5, C.ink, { bold: true });
    text(s, r[1], 9.98, y + 0.42, 1.95, 0.22, 7.8, C.muted);
  });
  rect(s, 9.25, 6.36, 3.1, 0.42, 'FFF7EA', 0.1, C.line);
  text(s, '固定巡視  →  事件驅動', 9.46, 6.45, 2.68, 0.18, 8.5, C.red, { bold: true, align: 'center' });
}

// 4 — Solution stack
{
  const s = addSlide(); title(s, 'Solution', '無接觸感測，從床邊一路到決策端', '同一條資料鏈，同時支援即時監測、AI 判定、事件警報與跨平台串接。');
  const layers = [
    { x: 0.7, n: '01', t: '感測', d: '無接觸取得\n生命表徵與熱像', c: C.cyan },
    { x: 3.25, n: '02', t: '處理', d: '邊緣運算、\n資料清洗與儲存', c: C.blue },
    { x: 5.8, n: '03', t: '理解', d: '人形、翻身拍背、\n床緣與異常判定', c: C.magenta },
    { x: 8.35, n: '04', t: '行動', d: '護理站警報、\n趨勢與事件紀錄', c: C.orange },
    { x: 10.9, n: '05', t: '串接', d: 'API、報表、\n指定平台上傳', c: C.green },
  ];
  layers.forEach((v, i) => {
    rect(s, v.x, 2.05, 1.78, 3.35, C.panel, 0.16, C.line);
    slideCircle(s, v.x + 0.54, 2.38, 0.7, v.c, v.n);
    text(s, v.t, v.x + 0.22, 3.38, 1.35, 0.4, 16, C.ink, { bold: true, align: 'center' });
    text(s, v.d, v.x + 0.18, 4.05, 1.42, 0.72, 9.4, C.muted, { align: 'center', breakLine: true, valign: 'top' });
    if (i < layers.length - 1) {
      s.addShape(pptx.ShapeType.chevron, { x: v.x + 1.88, y: 3.35, w: 0.45, h: 0.62, fill: { color: C.line }, line: { color: C.line } });
    }
  });
  text(s, '一個感測入口，形成可觀察、可解釋、可串接的照護資料資產。', 1.05, 5.92, 11.2, 0.55, 17, C.ink, { bold: true, align: 'center' });
}

function slideCircle(slide, x, y, d, color, label) {
  slide.addShape(pptx.ShapeType.ellipse, { x, y, w: d, h: d, fill: { color, transparency: 84 }, line: { color, width: 1.4 } });
  text(slide, label, x, y + 0.02, d, d - 0.04, 12, color, { bold: true, align: 'center' });
}

// 5 — Vital signs
{
  const s = addSlide(); title(s, 'Contactless sensing', '不接觸，也能持續掌握關鍵生命表徵', '把量測從一次性的動作，轉成連續、可追蹤的資訊。');
  const features = [
    ['心率 / 心跳', 'BPM', '掌握即時心率與心律狀態', C.red],
    ['呼吸率', 'RPM', '連續觀察呼吸頻率與變化', C.cyan],
    ['體表溫度', '°C', '非侵入式取得額頭皮膚表徵', C.orange],
  ];
  features.forEach((f, i) => {
    const x = 0.7 + i * 3.05;
    rect(s, x, 2.0, 2.75, 3.55, C.panel, 0.16, C.line);
    dot(s, x + 0.3, 2.35, f[3], 0.14);
    text(s, f[0], x + 0.3, 2.75, 2.1, 0.42, 15, C.ink, { bold: true });
    text(s, f[1], x + 0.3, 3.35, 2.1, 0.48, 22, f[3], { bold: true });
    text(s, f[2], x + 0.3, 4.28, 2.05, 0.65, 10, C.muted, { valign: 'top' });
  });
  rect(s, 9.88, 2.0, 2.74, 3.55, 'F8E9D3', 0.16, C.cyan);
  text(s, '無接觸的意義', 10.18, 2.38, 2.05, 0.38, 14, C.cyan2, { bold: true });
  const benefits = ['降低配戴不適', '減少逐次操作', '適合長時間觀察', '支援事件前後追溯'];
  benefits.forEach((b, i) => { dot(s, 10.2, 3.2 + i * 0.52, C.green, 0.09); text(s, b, 10.45, 3.12 + i * 0.52, 1.65, 0.28, 9.5, C.ink, { bold: true }); });
  footerNote(s, '體表溫度不等同核心體溫；產品定位為照護監測與決策輔助，非侵入式醫療診斷。');
}

// 6 — AI care intelligence
{
  const s = addSlide(); title(s, 'AI care intelligence', '不只量數值，也理解床邊正在發生什麼', '感測序列將姿態、動作與空間風險轉換成可用事件。');
  addCareEventCard(s, 0.62, 1.88, 3.8, 3.25, '01', '姿態與人在床', C.cyan, 'presence');
  addCareEventCard(s, 4.77, 1.88, 3.8, 3.25, '02', '翻身／拍背辨識', C.orange, 'care');
  addCareEventCard(s, 8.92, 1.88, 3.8, 3.25, '03', '床緣與安全區', C.magenta, 'edge');
  pill(s, '姿態與人在床', 1.32, 5.38, 2.4, C.cyan);
  pill(s, '翻身／拍背辨識', 5.47, 5.38, 2.4, C.orange);
  pill(s, '動作與遮擋容錯', 9.62, 5.38, 2.4, C.magenta);
  text(s, '加上床緣點位後，可同步判斷「安全」或「接觸床緣」，讓照護事件與風險在同一畫面呈現。', 1.0, 6.15, 11.3, 0.48, 13, C.ink, { bold: true, align: 'center' });
}

// 7 — Alert workflow
{
  const s = addSlide(); title(s, 'From signal to action', '警報不是多一個聲音，而是更清楚的優先順序', '把感測結果轉成狀態、事件、通知與可追溯紀錄。');
  const steps = [
    ['01', '持續偵測', '生命表徵、姿態、人在床'],
    ['02', 'AI 判讀', '異常、翻身拍背、床緣'],
    ['03', '即時提示', '床位燈號、警報、摘要'],
    ['04', '事件留痕', '時間、數值、判定與狀態'],
  ];
  steps.forEach((st, i) => {
    const x = 0.82 + i * 3.1;
    rect(s, x, 2.2, 2.45, 2.35, i === 2 ? 'F7DFC3' : C.panel, 0.18, i === 2 ? C.orange : C.line);
    text(s, st[0], x + 0.25, 2.47, 0.5, 0.3, 10, i === 2 ? C.orange : C.cyan, { bold: true });
    text(s, st[1], x + 0.25, 3.0, 1.95, 0.38, 16, C.ink, { bold: true });
    text(s, st[2], x + 0.25, 3.63, 1.95, 0.48, 9.3, C.muted, { valign: 'top' });
    if (i < 3) s.addShape(pptx.ShapeType.chevron, { x: x + 2.56, y: 3.03, w: 0.38, h: 0.6, fill: { color: C.line }, line: { color: C.line } });
  });
  rect(s, 1.25, 5.15, 10.82, 0.95, 'F8E9D3', 0.18, C.cyan);
  text(s, '異常先被看見  →  事件更快被理解  →  處置更容易被追蹤', 1.5, 5.38, 10.3, 0.45, 17, C.ink, { bold: true, align: 'center' });
}

// 8 — Dashboard
{
  const s = addSlide(); title(s, 'Command center', '一眼掌握整層狀態，點選即看單床細節', '床位狀態、警示燈、生命表徵與歷史趨勢集中在同一工作畫面。');
  addInterfaceScreenshot(s, A.commandCenter, 0.62, 1.78, 8.55, 4.92);
  const callouts = [
    ['全床位總覽', '快速辨識在線、離床與警示床位', C.cyan],
    ['單床詳情', '查看最新數值、判定與歷史時間軸', C.orange],
    ['即時燈號', '心率、心律、溫度、呼吸狀態', C.green],
    ['事件追溯', '依時間範圍回看趨勢與照護情境', C.magenta],
  ];
  callouts.forEach((c, i) => {
    const y = 1.92 + i * 1.15;
    rect(s, 9.55, y, 2.78, 0.92, C.panel, 0.12, C.line);
    dot(s, 9.82, y + 0.23, c[2], 0.1);
    text(s, c[0], 10.08, y + 0.12, 1.95, 0.28, 11, C.ink, { bold: true });
    text(s, c[1], 10.08, y + 0.46, 1.95, 0.26, 7.8, C.muted);
  });
}

// 9 — Architecture
{
  const s = addSlide(); title(s, 'AIoT architecture', '從感測器到平台，資料鏈可部署、可擴充', '即時資料與歷史資料分流，兼顧反應速度、追溯能力與外部整合。');
  const nodes = [
    { x: 0.65, w: 2.05, t: '床邊感測器', d: '生命表徵\n無接觸感測矩陣', c: C.cyan },
    { x: 3.05, w: 2.05, t: 'Go 接收服務', d: '資料驗證\n即時事件', c: C.blue },
    { x: 5.45, w: 2.05, t: '資料層', d: 'PostgreSQL\nRedis', c: C.magenta },
    { x: 7.85, w: 2.05, t: 'AI 分析', d: '人形／照護\n床緣／警報', c: C.orange },
    { x: 10.25, w: 2.45, t: '應用與串接', d: '護理站／API\n報表／外部平台', c: C.green },
  ];
  nodes.forEach((n, i) => {
    rect(s, n.x, 2.3, n.w, 2.18, C.panel, 0.16, n.c);
    slideCircle(s, n.x + (n.w - 0.62) / 2, 2.62, 0.62, n.c, String(i + 1).padStart(2, '0'));
    text(s, n.t, n.x + 0.14, 3.42, n.w - 0.28, 0.32, 12, C.ink, { bold: true, align: 'center' });
    text(s, n.d, n.x + 0.14, 3.86, n.w - 0.28, 0.42, 8.8, C.muted, { align: 'center', breakLine: true });
    if (i < nodes.length - 1) s.addShape(pptx.ShapeType.chevron, { x: n.x + n.w + 0.08, y: 3.03, w: 0.27, h: 0.6, fill: { color: C.line }, line: { color: C.line } });
  });
  const tags = [['即時', 'Redis 保存熱像與分析狀態', C.orange], ['歷史', 'PostgreSQL 保存可查詢量測', C.blue], ['開放', 'REST API 與報表輸出', C.green]];
  tags.forEach((t, i) => {
    const x = 1.15 + i * 4.05;
    rect(s, x, 5.22, 3.62, 0.78, 'F8E9D3', 0.14, C.line);
    text(s, t[0], x + 0.2, 5.4, 0.62, 0.25, 9, t[2], { bold: true });
    text(s, t[1], x + 0.9, 5.35, 2.45, 0.32, 8.4, C.ink, { bold: true });
  });
}

// 10 — Integration
{
  const s = addSlide(); title(s, 'Open integration', '即時數據不被鎖住，可送到指定平台', '以 API 或報表方式，銜接院內系統、照護平台、資料倉儲與管理儀表板。');
  rect(s, 0.65, 1.92, 5.45, 4.55, '5A3A2C', 0.16, C.line);
  text(s, 'REST API', 0.98, 2.25, 1.3, 0.32, 12, C.cyan, { bold: true });
  const code = `GET /api/ward/sensors/0201_04/history\n\n{\n  "heart_rate": 72,\n  "breath_rate": 16,\n  "surface_temperature": 31.6,\n  "turning_care": true,\n  "edge_contact": false,\n  "timestamp": "2026-07-17T15:00:00+08:00"\n}`;
  text(s, code, 1.0, 2.75, 4.7, 3.2, 10.3, 'FFF3E4', { fontFace: 'Menlo', valign: 'top', breakLine: true });
  const outs = [
    ['即時 API', '查詢最新狀態、歷史趨勢與事件'],
    ['事件推送', '異常或照護事件可推送指定端點'],
    ['報表輸出', '依床位、時段與事件產出摘要'],
    ['平台整合', '串接 HIS、照護平台或資料湖'],
  ];
  outs.forEach((o, i) => {
    const y = 1.98 + i * 1.12;
    rect(s, 6.55, y, 5.68, 0.88, C.panel, 0.12, C.line);
    text(s, String(i + 1).padStart(2, '0'), 6.82, y + 0.18, 0.48, 0.3, 9, i === 1 ? C.orange : C.cyan, { bold: true });
    text(s, o[0], 7.47, y + 0.12, 1.2, 0.3, 11.5, C.ink, { bold: true });
    text(s, o[1], 8.8, y + 0.12, 3.05, 0.42, 9, C.muted, { valign: 'top' });
  });
}

// 11 — Stakeholder value
{
  const s = addSlide(); title(s, 'Business value', '同一套系統，回應不同角色的目標', '從第一線效率、管理可視性到 IT 整合，形成可持續擴充的照護基礎。');
  const cols = [
    { x: 0.62, t: '照護人員', c: C.cyan, q: '少做重複量測，\n更快看見需要介入的床位', list: ['即時狀態集中', '事件自動留痕', '減少人工抄錄'] },
    { x: 3.72, t: '機構管理者', c: C.orange, q: '把照護品質與流程，\n轉成可觀察資訊', list: ['風險與警報摘要', '照護事件追溯', '導入成效可管理'] },
    { x: 6.82, t: '資訊團隊', c: C.blue, q: '標準介面整合，\n降低資料孤島與維運成本', list: ['API 優先架構', 'Redis＋PostgreSQL', '模組化部署'] },
    { x: 9.92, t: '合作平台', c: C.green, q: '快速取得即時數據，\n擴充既有產品服務', list: ['資料與事件串接', '報表與儀表板', '可客製輸出'] },
  ];
  cols.forEach((c) => {
    rect(s, c.x, 1.9, 2.78, 4.65, C.panel, 0.16, C.line);
    text(s, c.t, c.x + 0.28, 2.22, 2.15, 0.36, 14, c.c, { bold: true });
    text(s, c.q, c.x + 0.28, 2.95, 2.15, 0.88, 13, C.ink, { bold: true, breakLine: true, valign: 'top' });
    line(s, c.x + 0.28, 4.05, 2.1, 0, C.line, 1);
    c.list.forEach((v, i) => { dot(s, c.x + 0.3, 4.48 + i * 0.52, c.c, 0.08); text(s, v, c.x + 0.55, 4.39 + i * 0.52, 1.9, 0.26, 9, C.muted, { bold: true }); });
  });
}

// 12 — Commercial model
{
  const s = addSlide(); title(s, 'Go-to-market', '從單點驗證到規模部署，降低導入風險', '以場域驗證、分階段擴充與介面整合，建立可複製的商業導入模式。');
  const phases = [
    ['01', '場域盤點', '床位、網路、警報流程\n與串接需求確認', C.cyan],
    ['02', '小規模 Pilot', '校準量測與 AI 門檻\n驗證照護流程', C.blue],
    ['03', '樓層部署', '設備、儀表板、告警\n與人員教育', C.orange],
    ['04', '平台整合', 'API、報表、事件推送\n與營運優化', C.green],
  ];
  phases.forEach((p, i) => {
    const x = 0.7 + i * 3.1;
    rect(s, x, 1.98, 2.6, 2.25, C.panel, 0.15, p[3]);
    text(s, p[0], x + 0.24, 2.24, 0.52, 0.3, 10, p[3], { bold: true });
    text(s, p[1], x + 0.24, 2.78, 2.0, 0.34, 15, C.ink, { bold: true });
    text(s, p[2], x + 0.24, 3.38, 2.0, 0.52, 9, C.muted, { breakLine: true, valign: 'top' });
  });
  text(s, '商業組合', 0.72, 4.75, 1.1, 0.32, 10, C.cyan, { bold: true });
  const models = [['硬體設備', '感測器與邊緣設備'], ['軟體訂閱', '監測、AI 分析與管理介面'], ['整合服務', 'API、報表與指定平台串接']];
  models.forEach((m, i) => {
    const x = 2.05 + i * 3.55;
    rect(s, x, 4.65, 3.15, 1.18, 'F8E9D3', 0.15, C.line);
    text(s, m[0], x + 0.25, 4.88, 2.55, 0.3, 12, i === 0 ? C.orange : i === 1 ? C.cyan : C.green, { bold: true });
    text(s, m[1], x + 0.25, 5.3, 2.55, 0.25, 8.5, C.muted);
  });
  text(s, '依床位數、功能模組、資料留存與整合範圍規劃方案。', 2.05, 6.15, 9.8, 0.3, 10, C.ink, { bold: true, align: 'center' });
}

// 13 — Close
{
  const s = addSlide();
  rect(s, 7.38, 0.48, 5.44, 6.25, 'F4E3C9', 0.2, C.line);
  addProductImage(s, A.productFront, 7.62, 1.1, 4.96, 5.05);
  pill(s, 'NEXT STEP', 0.65, 0.72, 1.55, C.red, 'F8E9D3');
  text(s, '讓照護人員，\n把時間留給照護。', 0.65, 1.48, 6.15, 1.35, 32, C.ink, { bold: true, breakLine: true, valign: 'top' });
  text(s, '從一個床位開始驗證，\n建立可擴充到整層、整院與跨平台的照護感知能力。', 0.68, 3.28, 5.7, 0.9, 14, C.muted, { breakLine: true, valign: 'top' });
  rect(s, 0.66, 4.72, 5.85, 1.15, C.panel, 0.16, C.cyan);
  text(s, '建議下一步', 0.94, 4.96, 1.25, 0.25, 9, C.cyan, { bold: true });
  text(s, '場域訪談  →  Pilot 規格  →  量測與流程驗證', 0.94, 5.34, 5.0, 0.3, 12, C.ink, { bold: true });
  text(s, 'CARE SENSE  ·  CONTACTLESS CARE INTELLIGENCE', 0.68, 6.42, 5.8, 0.25, 8, C.muted, { charSpacing: 1.2 });
}

for (const slide of pptx._slides) {
  warnIfSlideHasOverlaps(slide, pptx);
  warnIfSlideElementsOutOfBounds(slide, pptx);
}

pptx.writeFile({ fileName: 'care_sense_product_overview_zh-TW.pptx' });
