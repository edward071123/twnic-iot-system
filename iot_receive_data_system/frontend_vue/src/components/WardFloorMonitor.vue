<template>
  <div ref="rootEl" class="layout">
    <section class="left card">
      <div class="head" :class="{ 'head-engineer': props.engineerMode }">
        <div class="head-main">
          <h1>
            <span id="current-floor-title">2F</span>
            <span class="ward-title-text">{{ t('ward.title') }}</span>
            <span class="status-inline" :aria-label="t('ward.legend.label')">
              <span class="legend-box">
                <span class="legend-item"><span>{{ t('ward.fields.heartRate') }}</span><span class="legend-state"><i class="dot hr-ok"></i><small>{{ t('ward.legend.normal') }}</small></span><span class="legend-state"><i class="dot hr-warn"></i><small>{{ t('ward.legend.warning') }}</small></span></span>
                <span class="legend-item"><span>{{ t('ward.fields.rhythm') }}</span><span class="legend-state"><i class="dot rhythm-ok"></i><small>{{ t('ward.legend.normal') }}</small></span><span class="legend-state"><i class="dot rhythm-warn"></i><small>{{ t('ward.legend.warning') }}</small></span></span>
                <span class="legend-item"><span>{{ t('ward.fields.temperature') }}</span><span class="legend-state"><i class="dot temp-ok"></i><small>{{ t('ward.legend.normal') }}</small></span><span class="legend-state"><i class="dot temp-warn"></i><small>{{ t('ward.legend.warning') }}</small></span></span>
                <span class="legend-item"><span>{{ t('ward.fields.breath') }}</span><span class="legend-state"><i class="dot breath-ok"></i><small>{{ t('ward.legend.normal') }}</small></span><span class="legend-state"><i class="dot breath-warn"></i><small>{{ t('ward.legend.warning') }}</small></span></span>
                <span class="legend-item legend-bed-state"><span>{{ t('ward.legend.bedState') }}</span><span class="legend-state"><i class="swatch bed-online-in"></i><small>{{ t('ward.legend.onlineInBed') }}</small></span><span class="legend-state"><i class="swatch bed-online-out"></i><small>{{ t('ward.legend.onlineOutOfBed') }}</small></span><span class="legend-state"><i class="swatch bed-offline"></i><small>{{ t('ward.legend.offline') }}</small></span></span>
              </span>
            </span>
          </h1>
          <div v-if="props.engineerMode" class="engineer-logout-row">
            <button
              class="engineer-logout"
              type="button"
              :aria-label="t('ward.engineer.logout')"
              @click="logoutEngineerMode"
            >
              {{ t('ward.engineer.logout') }}
            </button>
          </div>
        </div>
        <div class="actions">
          <div class="language-inline" :aria-label="t('ward.language.switch')">
            <button id="language-prev" class="language-nav" type="button" :aria-label="t('ward.language.prev')" :title="t('ward.language.prev')">‹</button>
            <span class="language-window">
              <span id="language-select" class="language-switch" role="group" :aria-label="t('ward.language.switch')">
                <button class="language-option" type="button" data-lang="en" aria-pressed="false">{{ t('ward.language.en') }}</button>
                <button class="language-option active" type="button" data-lang="zh-TW" aria-pressed="true">{{ t('ward.language.zh') }}</button>
                <button class="language-option" type="button" data-lang="vi" aria-pressed="false">{{ t('ward.language.vi') }}</button>
              </span>
            </span>
            <button id="language-next" class="language-nav" type="button" :aria-label="t('ward.language.next')" :title="t('ward.language.next')">›</button>
          </div>
          <div class="floor-inline" :aria-label="t('ward.floor.switch')">
            <button id="floor-prev" class="floor-nav" type="button" :aria-label="t('ward.floor.prev')" :title="t('ward.floor.prev')">‹</button>
            <span class="floor-window">
              <span id="floor-select" class="floor-switch" role="group" :aria-label="t('ward.floor.select')"></span>
            </span>
            <button id="floor-next" class="floor-nav" type="button" :aria-label="t('ward.floor.next')" :title="t('ward.floor.next')">›</button>
          </div>
        </div>
      </div>

      <div class="plan-wrap">
        <svg id="plan" viewBox="0 0 1110 840" preserveAspectRatio="none" xmlns="http://www.w3.org/2000/svg" :aria-label="t('ward.floor.planLabel', { floor: 2 })">
          <g id="plan-canvas" transform="matrix(1.305882,0,0,1.135135,-143.647,-79.459)">
          <path class="outline" d="M110 70 H960 V810 H110 Z" />

          <path class="corridor" d="M220 320 H960 V560 H430 V760 H220 Z" />

          <g id="rooms-layer"></g>
          <g id="nurse-station">
            <rect class="nurse-shape" x="620" y="350" width="245" height="150" />
            <text id="nurse-label" class="nurse-label" x="742.5" y="433" text-anchor="middle">{{ t('ward.nurseStation') }}</text>
          </g>

          <g id="beds-layer"></g>
          </g>
        </svg>
        <div v-if="props.showThermal" class="thermal-panel" :aria-label="t('ward.thermal.label')">
          <div class="thermal-head">
            <strong>{{ t('ward.thermal.title') }}</strong>
            <span id="thermal-sensor">--</span>
          </div>
          <canvas id="thermal-canvas" width="560" height="390"></canvas>
          <div id="thermal-range" class="thermal-range">{{ t('ward.thermal.noData') }}</div>
          <div class="thermal-timeline">
            <input id="thermal-history-slider" type="range" min="0" max="0" value="0" disabled />
            <div class="thermal-time-row">
              <span id="thermal-time-label">{{ t('ward.thermal.noData') }}</span>
              <span id="thermal-frame-count">0 / 0</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <aside class="right card">
      <div>
        <div class="selection-head">
          <h2 id="sel-title">{{ t('ward.selection.noBed') }}</h2>
          <div class="summary" id="summary">{{ t('ward.summary.loading') }}</div>
        </div>
        <div class="clock" id="clock">--:--:--</div>
      </div>
      <div id="sel-meta">
        <div class="meta-col">
          <div class="kv"><b>{{ t('ward.fields.patient') }}</b><span>{{ t('ward.selection.pickAnyBed') }}</span></div>
          <div class="kv"><b>{{ t('ward.fields.rhythm') }}</b><span>-</span></div>
          <div class="kv"><b>{{ t('ward.fields.temperature') }}</b><span>-</span></div>
        </div>
        <div class="meta-col">
          <div class="kv"><b>{{ t('ward.fields.heartRate') }}</b><span>-</span></div>
          <div class="kv"><b>{{ t('ward.fields.breath') }}</b><span>-</span></div>
          <div class="kv"><b>{{ t('ward.fields.device') }}</b><span>-</span></div>
        </div>
      </div>
      <div class="range-panel">
        <el-config-provider :locale="elementLocale">
          <div class="range-grid">
            <label class="range-field">
              <span>{{ t('ward.range.startDateTime') }}</span>
              <el-date-picker
                v-model="rangeStartPicker"
                type="datetime"
                :format="dateTimeDisplayFormat"
                :time-format="'HH:mm'"
                :editable="false"
                :clearable="true"
                :teleported="false"
                :placeholder="t('ward.range.startDateTime')"
              />
            </label>
            <label class="range-field">
              <span>{{ t('ward.range.endDateTime') }}</span>
              <el-date-picker
                v-model="rangeEndPicker"
                type="datetime"
                :format="dateTimeDisplayFormat"
                :time-format="'HH:mm'"
                :editable="false"
                :clearable="true"
                :teleported="false"
                :placeholder="t('ward.range.endDateTime')"
              />
            </label>
          </div>
        </el-config-provider>
        <div class="range-actions">
          <button id="apply-range" class="alt" type="button">{{ t('ward.range.apply') }}</button>
          <button id="quick-5m" class="alt" type="button">{{ t('ward.range.quick5m') }}</button>
          <button id="quick-1h" class="alt" type="button">{{ t('ward.range.quick1h') }}</button>
          <button id="quick-1d" class="alt" type="button">{{ t('ward.range.quick1d') }}</button>
          <button id="quick-3d" class="alt" type="button">{{ t('ward.range.quick3d') }}</button>
          <button id="quick-7d" class="alt" type="button">{{ t('ward.range.quick7d') }}</button>
          <button id="clear-range" class="alt" type="button">{{ t('ward.range.clear') }}</button>
        </div>
      </div>
      <div class="charts">
        <div class="chart-card">
          <div class="chart-head">
            <span>{{ t('ward.fields.temperature') }}</span>
            <span id="temp-now" class="chart-now">--</span>
          </div>
          <svg id="temp-chart" class="chart-svg" viewBox="0 0 300 126" preserveAspectRatio="none"></svg>
        </div>
        <div class="chart-card">
          <div class="chart-head">
            <span>{{ t('ward.fields.heartRate') }}</span>
            <span id="hr-now" class="chart-now">--</span>
          </div>
          <svg id="hr-chart" class="chart-svg" viewBox="0 0 300 126" preserveAspectRatio="none"></svg>
        </div>
        <div class="chart-card">
          <div class="chart-head">
            <span>{{ t('ward.fields.breath') }}</span>
            <span id="breath-now" class="chart-now">--</span>
          </div>
          <svg id="breath-chart" class="chart-svg" viewBox="0 0 300 126" preserveAspectRatio="none"></svg>
        </div>
      </div>
    </aside>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import 'dayjs/locale/en'
import 'dayjs/locale/vi'
import 'dayjs/locale/zh-tw'
import enLocale from 'element-plus/es/locale/lang/en'
import viLocale from 'element-plus/es/locale/lang/vi'
import zhTwLocale from 'element-plus/es/locale/lang/zh-tw'
import { fetchSensorHistory, fetchSensorThermalFrame, fetchSensorThermalLatest, fetchSensorThermalTimeline, fetchWardFloorOverview, fetchWardFloors, openWardFloorStream } from '../api/ward'
import { baseBedNumberMap, baseRoomGridOverrides, buildRoomDefs, floorBedNumberMaps, floorRoomGridOverrides, getFloorConfig } from '../config/wardLayouts'
import { clearEngineerAuthed } from '../auth/engineerAuth'

let cleanup = () => {}
const rootEl = ref(null)
const router = useRouter()
const props = defineProps({
  showThermal: {
    type: Boolean,
    default: true,
  },
  engineerMode: {
    type: Boolean,
    default: false,
  },
})
const { t, locale } = useI18n()
const rangeStartPicker = ref(null)
const rangeEndPicker = ref(null)

const localeToDayjs = {
  'zh-TW': 'zh-tw',
  en: 'en',
  vi: 'vi',
}
const localeToElement = {
  'zh-TW': zhTwLocale,
  en: enLocale,
  vi: viLocale,
}
const localeToDateTimeFormat = {
  'zh-TW': 'YYYY/MM/DD HH:mm',
  en: 'MM/DD/YYYY HH:mm',
  vi: 'DD/MM/YYYY HH:mm',
}

const elementLocale = computed(() => localeToElement[locale.value] || zhTwLocale)
const dateTimeDisplayFormat = computed(() => localeToDateTimeFormat[locale.value] || localeToDateTimeFormat['zh-TW'])
dayjs.locale(localeToDayjs[locale.value] || localeToDayjs['zh-TW'])

function logoutEngineerMode() {
  clearEngineerAuthed()
  router.replace('/engineer/login')
}

onMounted(async () => {
      await nextTick()
      const byId = (id) => rootEl.value?.querySelector(`#${id}`) || document.getElementById(id);
      let selectedFloor = 2;
      let availableFloors = [{ floor: 2, label: '2F' }, { floor: 3, label: '3F' }];
      let roomDefs = buildRoomDefs(selectedFloor);
      let customBedNumberMap = getFloorConfig(floorBedNumberMaps, selectedFloor);
      let roomGridOverrides = getFloorConfig(floorRoomGridOverrides, selectedFloor);
      let selectedLanguage = locale.value;
  
      const surnames = ['王', '李', '陳', '林', '張', '黃', '吳', '劉', '蔡', '楊', '許', '鄭', '謝', '洪', '郭'];
      const given = ['雅雯', '志明', '俊廷', '思妤', '家豪', '柏翰', '佳蓉', '泓宇', '怡君', '建宏', '佩穎', '承恩'];
  
      const beds = [];
      let selectedBedId = null;
      let selectedRoomId = null;
      let timer = null;
      let rangeStartMs = null;
      let rangeEndMs = null;
      let apiMode = false;
      let streamConnected = false;
      let wardStream = null;
      let lastOverviewFetchMs = 0;
      let historyRequestToken = 0;
      let lastSelectedHistoryFetchMs = 0;
      let selectedHistoryInFlight = false;
      let lastSelectedThermalFetchMs = 0;
      let selectedThermalInFlight = false;
      let thermalTimelineIndex = null;
      let thermalFrameRequestKey = '';
      const updatedClassTimers = new Map();
  
      const maxHistoryLength = 12000;
      const liveChartWindowMs = 5 * 60 * 1000;
      const overviewPollMs = 500;
      const selectedHistoryPollMs = 500;
      const selectedThermalPollMs = 500;
      const deviceOfflineMs = Number(import.meta.env.VITE_DEVICE_OFFLINE_MS || 5000);
    const roomsLayer = byId('rooms-layer');
    const bedsLayer = byId('beds-layer');
    if (!roomsLayer || !bedsLayer) {
      console.error('rooms-layer or beds-layer not found, cannot render ward plan');
      return;
    }
  
      const createSvg = (tag, attrs = {}) => {
        const n = document.createElementNS('http://www.w3.org/2000/svg', tag);
        for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, String(v));
        return n;
      };
  
      const seeded = (n) => {
        const x = Math.sin(n * 83.71) * 10000;
        return x - Math.floor(x);
      };

      const infernoStops = [
        { t: 0.0, c: [0, 0, 4] },
        { t: 0.13, c: [31, 12, 72] },
        { t: 0.25, c: [85, 15, 109] },
        { t: 0.38, c: [136, 34, 106] },
        { t: 0.5, c: [186, 54, 85] },
        { t: 0.63, c: [224, 85, 61] },
        { t: 0.75, c: [243, 131, 36] },
        { t: 0.88, c: [251, 180, 20] },
        { t: 1.0, c: [252, 255, 164] },
      ];
      const mlx90640Rows = 24;
      const mlx90640Cols = 32;
      // Align with doc/GYMCU90640.py: fliplr is optional and disabled by default.
      const thermalFlipLeftRight = false;
      const infernoLutSize = 256;

      function lerp(a, b, ratio) {
        return a + (b - a) * ratio;
      }

      function infernoColorFromStops(tValue) {
        for (let i = 1; i < infernoStops.length; i += 1) {
          const prev = infernoStops[i - 1];
          const next = infernoStops[i];
          if (tValue <= next.t) {
            const ratio = (tValue - prev.t) / (next.t - prev.t || 1);
            return [
              Math.round(lerp(prev.c[0], next.c[0], ratio)),
              Math.round(lerp(prev.c[1], next.c[1], ratio)),
              Math.round(lerp(prev.c[2], next.c[2], ratio)),
            ];
          }
        }
        return infernoStops[infernoStops.length - 1].c.slice();
      }
      
      const infernoLut = Array.from({ length: infernoLutSize }, (_, index) => {
        const tValue = index / (infernoLutSize - 1);
        return infernoColorFromStops(tValue);
      });

      function infernoColor(normalized) {
        const tValue = Math.max(0, Math.min(1, normalized));
        const lutIndex = Math.round(tValue * (infernoLutSize - 1));
        return infernoLut[lutIndex];
      }

      function parseTemperatureFrame(raw) {
        if (raw === undefined || raw === null) return null;
        let source = raw;
        if (typeof source === 'string') {
          try {
            source = JSON.parse(source);
          } catch (error) {
            return null;
          }
        }
        if (!Array.isArray(source) || source.length === 0) return null;
        const values = source.map((value) => Number(value));
        if (values.some((value) => !Number.isFinite(value))) return null;
        if (values.length === mlx90640Rows * mlx90640Cols) {
          return { values, rows: mlx90640Rows, cols: mlx90640Cols };
        }
        const side = Math.sqrt(values.length);
        if (!Number.isInteger(side) || side < 4) return null;
        return { values, rows: side, cols: side };
      }

      function dynamicThermalScale(values) {
        const pixelMin = Math.min(...values);
        const pixelMax = Math.max(...values);
        const vmin = pixelMin;
        const vmax = pixelMax;
        return { pixelMin, pixelMax, vmin, vmax };
      }

      function clampInt(value, min, max) {
        if (value < min) return min;
        if (value > max) return max;
        return value;
      }

      function cubicWeight(distance) {
        const a = -0.5; // Catmull-Rom bicubic kernel
        const t = Math.abs(distance);
        if (t <= 1) {
          return (a + 2) * t * t * t - (a + 3) * t * t + 1;
        }
        if (t < 2) {
          return a * t * t * t - 5 * a * t * t + 8 * a * t - 4 * a;
        }
        return 0;
      }

      function bicubicResizeImageData(srcImage, srcWidth, srcHeight, dstWidth, dstHeight) {
        const output = new ImageData(dstWidth, dstHeight);
        const src = srcImage.data;
        const dst = output.data;
        const srcXScale = srcWidth / dstWidth;
        const srcYScale = srcHeight / dstHeight;

        for (let y = 0; y < dstHeight; y += 1) {
          const sourceY = (y + 0.5) * srcYScale - 0.5;
          const baseY = Math.floor(sourceY);
          for (let x = 0; x < dstWidth; x += 1) {
            const sourceX = (x + 0.5) * srcXScale - 0.5;
            const baseX = Math.floor(sourceX);
            const outIndex = (y * dstWidth + x) * 4;

            for (let channel = 0; channel < 3; channel += 1) {
              let weighted = 0;
              let weightSum = 0;

              for (let ky = -1; ky <= 2; ky += 1) {
                const sampleY = clampInt(baseY + ky, 0, srcHeight - 1);
                const wy = cubicWeight(sourceY - (baseY + ky));
                if (wy === 0) continue;
                for (let kx = -1; kx <= 2; kx += 1) {
                  const sampleX = clampInt(baseX + kx, 0, srcWidth - 1);
                  const wx = cubicWeight(sourceX - (baseX + kx));
                  if (wx === 0) continue;
                  const weight = wx * wy;
                  const sampleIndex = (sampleY * srcWidth + sampleX) * 4 + channel;
                  weighted += src[sampleIndex] * weight;
                  weightSum += weight;
                }
              }

              const channelValue = weightSum === 0 ? 0 : weighted / weightSum;
              dst[outIndex + channel] = clampInt(Math.round(channelValue), 0, 255);
            }

            dst[outIndex + 3] = 255;
          }
        }

        return output;
      }

      function buildMockThermalFrame(baseTemp, seedBase) {
        const rows = mlx90640Rows;
        const cols = mlx90640Cols;
        const values = [];
        const hotX = cols * (0.25 + seeded(seedBase + 31) * 0.5);
        const hotY = rows * (0.25 + seeded(seedBase + 47) * 0.5);
        for (let y = 0; y < rows; y += 1) {
          for (let x = 0; x < cols; x += 1) {
            const distance = Math.hypot(x - hotX, y - hotY);
            const hotspot = Math.max(0, 1 - distance / 8) * (1.0 + seeded(seedBase + x * 7 + y * 11));
            const noise = (seeded(seedBase + x * 13 + y * 17) - 0.5) * 0.35;
            values.push(Number((baseTemp + hotspot + noise).toFixed(2)));
          }
        }
        return { values, rows, cols };
      }

      function getFilteredThermalHistory(bed) {
        const frames = bed?.thermalHistory || [];
        if (rangeStartMs === null && rangeEndMs === null) {
          return [];
        }
        return frames.filter((item) => (
          (rangeStartMs === null || item.ts >= rangeStartMs) &&
          (rangeEndMs === null || item.ts <= rangeEndMs)
        ));
      }

      function formatThermalTime(ts) {
        if (!ts) return t('ward.thermal.noData');
        return new Date(ts).toLocaleString(selectedLanguage, {
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit'
        });
      }

      function updateThermalTimeline(frames, selectedIndex, frameTs) {
        const slider = byId('thermal-history-slider');
        const timeLabel = byId('thermal-time-label');
        const countLabel = byId('thermal-frame-count');
        if (!slider || !timeLabel || !countLabel) return;

        if (rangeStartMs === null && rangeEndMs === null) {
          slider.min = '0';
          slider.max = '0';
          slider.value = '0';
          slider.disabled = true;
          if (!frameTs) {
            timeLabel.textContent = t('ward.thermal.noData');
            countLabel.textContent = '0 / 0';
            return;
          }
          timeLabel.textContent = frameTs ? `${t('ward.thermal.live')}: ${formatThermalTime(frameTs)}` : t('ward.thermal.live');
          countLabel.textContent = t('ward.thermal.live');
          return;
        }

        if (!frames.length) {
          slider.min = '0';
          slider.max = '0';
          slider.value = '0';
          slider.disabled = true;
          timeLabel.textContent = frameTs ? formatThermalTime(frameTs) : t('ward.thermal.noData');
          countLabel.textContent = frameTs ? '1 / 1' : '0 / 0';
          return;
        }

        slider.disabled = frames.length <= 1;
        slider.min = '0';
        slider.max = String(frames.length - 1);
        slider.value = String(selectedIndex);
        timeLabel.textContent = formatThermalTime(frameTs);
        countLabel.textContent = `${selectedIndex + 1} / ${frames.length}`;
      }

      async function loadThermalFrameAtIndex(bed, index) {
        if (!bed?.sensorNumber || !Array.isArray(bed.thermalHistory)) return;
        const entry = bed.thermalHistory[index];
        if (!entry?.dataId || entry.frame?.values?.length) return;
        const requestKey = `${bed.sensorNumber}:${entry.dataId}`;
        if (thermalFrameRequestKey === requestKey) return;
        thermalFrameRequestKey = requestKey;
        try {
          const point = await fetchSensorThermalFrame(bed.sensorNumber, entry.dataId);
          if (thermalFrameRequestKey !== requestKey) return;
          const frame = parseTemperatureFrame(point?.temperature_json);
          if (!frame) return;
          entry.frame = frame;
          bed.thermalFrame = frame;
          renderThermalPanel();
        } catch (err) {
          console.warn('sensor thermal frame api unavailable', err);
        } finally {
          if (thermalFrameRequestKey === requestKey) thermalFrameRequestKey = '';
        }
      }

      function renderThermalPanel() {
        const canvas = byId('thermal-canvas');
        const sensorEl = byId('thermal-sensor');
        const rangeEl = byId('thermal-range');
        if (!canvas || !sensorEl || !rangeEl) return;
        const ctx = canvas.getContext('2d');
        if (!ctx) return;

        const bed = selectedBedId ? beds.find((item) => item.id === selectedBedId) : null;
        const thermalFrames = getFilteredThermalHistory(bed);
        const rangeActive = rangeStartMs !== null || rangeEndMs !== null;
        let selectedTimelineIndex = -1;
        let frameTs = null;
        let frame = rangeActive ? null : (bed?.thermalFrame || null);
        if (rangeActive && thermalFrames.length > 0) {
          selectedTimelineIndex = thermalTimelineIndex === null
            ? thermalFrames.length - 1
            : clampInt(thermalTimelineIndex, 0, thermalFrames.length - 1);
          frame = thermalFrames[selectedTimelineIndex].frame;
          frameTs = thermalFrames[selectedTimelineIndex].ts;
          if (!frame?.values?.length) {
            updateThermalTimeline(thermalFrames, selectedTimelineIndex, frameTs);
            sensorEl.textContent = bed?.sensorNumber || '--';
            rangeEl.textContent = t('ward.thermal.noData');
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = '#0a1d30';
            ctx.fillRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = '#6c8fb3';
            ctx.font = '12px "Noto Sans TC", sans-serif';
            ctx.textAlign = 'center';
            ctx.fillText(t('ward.thermal.loading'), canvas.width / 2, canvas.height / 2);
            loadThermalFrameAtIndex(bed, selectedTimelineIndex);
            return;
          }
        } else if (!rangeActive) {
          frameTs = bed?.latestTimestamp || null;
        }
        if (!bed || !frame || !frame.values?.length) {
          ctx.clearRect(0, 0, canvas.width, canvas.height);
          ctx.fillStyle = '#0a1d30';
          ctx.fillRect(0, 0, canvas.width, canvas.height);
          ctx.fillStyle = '#6c8fb3';
          ctx.font = '12px "Noto Sans TC", sans-serif';
          ctx.textAlign = 'center';
          ctx.fillText(t('ward.thermal.noData'), canvas.width / 2, canvas.height / 2);
          sensorEl.textContent = '--';
          rangeEl.textContent = t('ward.thermal.noData');
          updateThermalTimeline([], -1, null);
          return;
        }
        updateThermalTimeline(thermalFrames, selectedTimelineIndex < 0 ? 0 : selectedTimelineIndex, frameTs);

        const { values, rows, cols } = frame;
        if (!Number.isInteger(rows) || !Number.isInteger(cols) || rows <= 0 || cols <= 0 || rows * cols !== values.length) {
          return;
        }
        const { pixelMin, pixelMax, vmin, vmax } = dynamicThermalScale(values);
        const image = new ImageData(cols, rows);
        const span = Math.max(0.001, vmax - vmin);

        for (let i = 0; i < values.length; i += 1) {
          const row = Math.floor(i / cols);
          const col = i % cols;
          const sourceCol = thermalFlipLeftRight ? cols - 1 - col : col;
          const sourceIndex = row * cols + sourceCol;
          const normalized = Math.max(0, Math.min(1, (values[sourceIndex] - vmin) / span));
          const [r, g, b] = infernoColor(normalized);
          const base = i * 4;
          image.data[base] = r;
          image.data[base + 1] = g;
          image.data[base + 2] = b;
          image.data[base + 3] = 255;
        }

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.fillStyle = '#0a1d30';
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        const srcRatio = cols / rows;
        const canvasRatio = canvas.width / canvas.height;
        let drawW = canvas.width;
        let drawH = canvas.height;
        let drawX = 0;
        let drawY = 0;
        if (srcRatio > canvasRatio) {
          drawH = canvas.width / srcRatio;
          drawY = (canvas.height - drawH) / 2;
        } else if (srcRatio < canvasRatio) {
          drawW = canvas.height * srcRatio;
          drawX = (canvas.width - drawW) / 2;
        }
        const drawWidth = Math.max(1, Math.round(drawW));
        const drawHeight = Math.max(1, Math.round(drawH));
        const drawLeft = Math.round(drawX);
        const drawTop = Math.round(drawY);
        const resizedImage = bicubicResizeImageData(image, cols, rows, drawWidth, drawHeight);
        ctx.putImageData(resizedImage, drawLeft, drawTop);

        sensorEl.textContent = bed.sensorNumber || '--';
        rangeEl.textContent = `${t('ward.thermal.min')}: ${pixelMin.toFixed(1)}°C  ${t('ward.thermal.max')}: ${pixelMax.toFixed(1)}°C`;
      }

      function syncPlanText() {
        byId('plan')?.setAttribute('aria-label', t('ward.floor.planLabel', { floor: selectedFloor }));
        const nurseLabel = byId('nurse-label');
        if (nurseLabel) {
          nurseLabel.textContent = t('ward.nurseStation');
          nurseLabel.setAttribute('font-size', selectedLanguage === 'vi' ? '18px' : selectedLanguage === 'en' ? '20px' : '28px');
        }
      }

      function renderRooms() {
        roomsLayer.innerHTML = '';
        roomDefs.forEach((room) => {
          const roomGroup = createSvg('g', { id: `room-${room.id}` });
          const roomRect = createSvg('rect', {
            class: 'room-shape',
            x: room.x,
            y: room.y,
            width: room.w,
            height: room.h,
          });
          const roomLabel = createSvg('text', {
            class: 'room-label',
            x: room.label?.x ?? room.x + room.w / 2,
            y: room.label?.y ?? room.y + room.h / 2,
          });
          if (room.label?.textAnchor) roomLabel.setAttribute('text-anchor', room.label.textAnchor);
          roomLabel.textContent = room.id;
          const door = createSvg('line', {
            class: 'door',
            x1: room.door.x1,
            y1: room.door.y1,
            x2: room.door.x2,
            y2: room.door.y2,
          });

          roomGroup.append(roomRect, roomLabel, door);
          roomGroup.addEventListener('click', (event) => {
            event.stopPropagation();
            selectedRoomId = room.id;
            if (!selectedBedId || !selectedBedId.startsWith(`${room.id}-`)) selectedBedId = null;
            syncSelection();
          });
          roomsLayer.appendChild(roomGroup);
        });
        syncPlanText();
      }

      function renderFloorOptions() {
        const floorSwitch = byId('floor-select');
        if (!floorSwitch) return;
        floorSwitch.innerHTML = '';
        const currentLabel = availableFloors.find((floor) => floor.floor === selectedFloor)?.label || `${selectedFloor}F`;
        const title = byId('current-floor-title');
        if (title) title.textContent = currentLabel;
        availableFloors.forEach((floor) => {
          const button = document.createElement('button');
          button.className = `floor-option ${floor.floor === selectedFloor ? 'active' : ''}`;
          button.type = 'button';
          button.dataset.floor = String(floor.floor);
          button.textContent = floor.label || `${floor.floor}F`;
          floorSwitch.appendChild(button);
        });
        updateFloorNavState();
        requestAnimationFrame(() => {
          floorSwitch.querySelector('.floor-option.active')?.scrollIntoView({
            behavior: 'smooth',
            inline: 'center',
            block: 'nearest'
          });
        });
      }

      function selectedFloorIndex() {
        return availableFloors.findIndex((floor) => floor.floor === selectedFloor);
      }

      function updateFloorNavState() {
        const idx = selectedFloorIndex();
        const prev = byId('floor-prev');
        const next = byId('floor-next');
        const canMove = availableFloors.length > 1 && idx >= 0;
        if (prev) prev.disabled = !canMove || idx === 0;
        if (next) next.disabled = !canMove || idx === availableFloors.length - 1;
      }

      function switchFloor(nextFloor) {
        if (!Number.isInteger(nextFloor) || nextFloor === selectedFloor) return;
        selectedFloor = nextFloor;
        renderFloorOptions();
        roomDefs = buildRoomDefs(selectedFloor);
        customBedNumberMap = getFloorConfig(floorBedNumberMaps, selectedFloor);
        roomGridOverrides = getFloorConfig(floorRoomGridOverrides, selectedFloor);
        thermalTimelineIndex = null;
        beds.length = 0;
        roomsLayer.innerHTML = '';
        bedsLayer.innerHTML = '';
        selectedBedId = null;
        selectedRoomId = null;
        apiMode = false;
        stopWardStream();
        lastOverviewFetchMs = 0;
        historyRequestToken += 1;
        renderRooms();
        renderBeds();
        updateRangeInputs();
        fetchOverview();
        startWardStream();
        syncSelection();
      }

      function stepFloor(direction) {
        const idx = selectedFloorIndex();
        if (idx < 0) return;
        const next = availableFloors[idx + direction];
        if (next) switchFloor(next.floor);
      }

      function setLanguage(lang) {
        if (!['zh-TW', 'en', 'vi'].includes(lang)) return;
        selectedLanguage = lang;
        locale.value = lang;
        dayjs.locale(localeToDayjs[lang] || localeToDayjs['zh-TW']);
        localStorage.setItem('locale', lang);
        document.documentElement.lang = lang;
        rootEl.value?.querySelectorAll('.language-option').forEach((button) => {
          const isActive = button.dataset.lang === selectedLanguage;
          button.classList.toggle('active', isActive);
          button.setAttribute('aria-pressed', String(isActive));
        });
        updateLanguageNavState();
        updateRangeInputs();
        requestAnimationFrame(() => {
          byId('language-select')?.querySelector('.language-option.active')?.scrollIntoView({
            behavior: 'smooth',
            inline: 'center',
            block: 'nearest'
          });
        });
        syncPlanText();
        updateSummary();
        syncSelection();
      }

      function selectedLanguageIndex() {
        return ['en', 'zh-TW', 'vi'].findIndex((lang) => lang === selectedLanguage);
      }

      function updateLanguageNavState() {
        const idx = selectedLanguageIndex();
        const prev = byId('language-prev');
        const next = byId('language-next');
        if (prev) prev.disabled = idx <= 0;
        if (next) next.disabled = idx < 0 || idx >= 2;
      }

      function stepLanguage(direction) {
        const languages = ['en', 'zh-TW', 'vi'];
        const idx = selectedLanguageIndex();
        if (idx < 0) return;
        const next = languages[idx + direction];
        if (next) setLanguage(next);
      }

      async function loadFloorOptions() {
        try {
          const floors = await fetchWardFloors();
          if (Array.isArray(floors) && floors.length > 0) {
            availableFloors = floors;
            selectedFloor = floors.some((floor) => floor.floor === selectedFloor) ? selectedFloor : floors[0].floor;
            roomDefs = buildRoomDefs(selectedFloor);
            customBedNumberMap = getFloorConfig(floorBedNumberMaps, selectedFloor);
            roomGridOverrides = getFloorConfig(floorRoomGridOverrides, selectedFloor);
          }
        } catch (err) {
          console.warn('ward floors api unavailable, fallback to default floors', err);
        }
        renderFloorOptions();
      }
  
      const randomName = (i) => {
        const s = surnames[Math.floor(seeded(i) * surnames.length)];
        const g = given[Math.floor(seeded(i + 11) * given.length)];
        return s + g;
      };
  
      const clamp = (v, min, max) => Math.max(min, Math.min(max, v));
      const pad2 = (n) => String(n).padStart(2, '0');

      const parsePickerMs = (value) => {
        if (value === '' || value === null || value === undefined) return null;
        if (value instanceof Date) {
          const ms = value.getTime();
          return Number.isNaN(ms) ? null : ms;
        }
        const parsed = Number(value);
        if (Number.isFinite(parsed)) return parsed;
        const ms = new Date(value).getTime();
        return Number.isNaN(ms) ? null : ms;
      };

      const updateRangeInputs = () => {
        rangeStartPicker.value = rangeStartMs === null ? null : new Date(rangeStartMs);
        rangeEndPicker.value = rangeEndMs === null ? null : new Date(rangeEndMs);
      };
  
      const level = (b) => {
        if (!b.deviceOnline) return 'warn';
        if (b.rhythm !== '竇性心律') return 'danger';
        if (b.temperature > 37.6 || b.temperature < 35.8 || b.heartRate > 110 || b.heartRate < 50) return 'warn';
        return 'ok';
      };

      const hasWarningLight = (bed) => (
        bed.heartRate < 50 ||
        bed.heartRate > 110 ||
        bed.rhythm !== '竇性心律' ||
        bed.temperature < 35.8 ||
        bed.temperature > 37.6 ||
        bed.breath < 10 ||
        bed.breath > 25
      );
  
      const rhythmKey = (value) => {
        switch (value) {
          case '竇性心律':
            return 'sinus';
          case '心房顫動':
            return 'afib';
          case '心搏過速':
            return 'tachycardia';
          case '心搏過緩':
            return 'bradycardia';
          default:
            return 'unknown';
        }
      };

      const rhythmText = (value) => t(`ward.rhythm.${rhythmKey(value)}`);
      const deviceStatusText = (isOnline) => (isOnline ? t('ward.status.online') : t('ward.status.offline'));
      const presenceStatusText = (isInBed) => (isInBed ? t('ward.legend.inBed') : t('ward.legend.outOfBed'));
      const pointIsInBed = (point) => {
        return true;
      };

      const hasFixedRange = () => rangeStartMs !== null || rangeEndMs !== null;
  
      const pushHistoryPoint = (bed, ts) => {
        bed.history.ts.push(ts);
        bed.history.temp.push(bed.temperature);
        bed.history.hr.push(bed.heartRate);
        bed.history.breath.push(bed.breath);
        Object.keys(bed.history).forEach((key) => {
          if (bed.history[key].length > maxHistoryLength) bed.history[key].shift();
        });
      };
  
      const resetHistory = (bed) => {
        bed.history = { ts: [], temp: [], hr: [], breath: [] };
        bed.thermalHistory = [];
      };

      const setHistoryFromPoints = (bed, points = []) => {
        bed.history = { ts: [], temp: [], hr: [], breath: [] };
        points.forEach((point) => {
          if (!pointIsInBed(point)) return;
          const ts = new Date(point?.timestamp).getTime();
          if (!Number.isFinite(ts)) return;
          const temp = Number(point.temperature ?? point.high_temperature);
          const hr = Number(point.heart_rate);
          const breath = Number(point.breath_rate);
          bed.history.ts.push(ts);
          bed.history.temp.push(Number.isFinite(temp) ? temp : NaN);
          bed.history.hr.push(Number.isFinite(hr) ? hr : NaN);
          bed.history.breath.push(Number.isFinite(breath) ? breath : NaN);
        });

        const latest = points[points.length - 1];
        if (latest) {
          const latestTs = new Date(latest.timestamp).getTime();
          if (Number.isFinite(latestTs)) bed.latestTimestamp = latestTs;
          bed.presence = true;
          const latestTemp = Number(latest.temperature ?? latest.high_temperature);
          const latestHr = Number(latest.heart_rate);
          const latestBreath = Number(latest.breath_rate);
          if (Number.isFinite(latestTemp)) bed.temperature = latestTemp;
          if (Number.isFinite(latestHr)) bed.heartRate = latestHr;
          if (Number.isFinite(latestBreath)) bed.breath = latestBreath;
          bed.rhythm = latest.rhythm || bed.rhythm;
        }
      };

      const setThermalTimeline = (bed, frames = []) => {
        bed.thermalHistory = frames
          .map((frame) => {
            const ts = new Date(frame.timestamp).getTime();
            const dataId = Number(frame.data_id);
            if (!Number.isFinite(ts) || !Number.isFinite(dataId)) return null;
            return { ts, dataId, frame: null };
          })
          .filter(Boolean);
      };

      const pushThermalHistoryFrame = (bed, ts, frame) => {
        if (!frame?.values?.length) return;
        const last = bed.thermalHistory[bed.thermalHistory.length - 1];
        if (last?.ts === ts) {
          last.frame = frame;
          return;
        }
        bed.thermalHistory.push({ ts, frame });
        if (bed.thermalHistory.length > maxHistoryLength) bed.thermalHistory.shift();
      };
  
      const applyPointToBed = (bed, point, options = {}) => {
        const { shouldPushHistory = true, updateLiveness = true } = options;
        if (!point) return;
        const ts = new Date(point.timestamp).getTime();
        if (Number.isNaN(ts)) return;
        bed.presence = true;
        const isInBed = bed.presence !== false;
        if (!isInBed) {
          bed.thermalFrame = null;
          if (updateLiveness) {
            bed.latestTimestamp = ts;
          }
          bed.lastUpdatedAt = Date.now();
          return;
        }
        bed.temperature = Number(point.temperature ?? point.high_temperature ?? bed.temperature);
        bed.heartRate = Number(point.heart_rate ?? bed.heartRate);
        bed.breath = Number(point.breath_rate ?? bed.breath);
        bed.rhythm = point.rhythm || bed.rhythm;
        const parsedFrame = parseTemperatureFrame(point.temperature_json);
        if (parsedFrame) {
          bed.thermalFrame = parsedFrame;
          if (shouldPushHistory) pushThermalHistoryFrame(bed, ts, parsedFrame);
        }
        if (updateLiveness) {
          bed.latestTimestamp = ts;
        }
        bed.lastUpdatedAt = Date.now();
        if (shouldPushHistory && bed.history.ts[bed.history.ts.length - 1] !== ts) {
          pushHistoryPoint(bed, ts);
        }
      };

      const refreshDeviceOnlineStates = () => {
        const now = Date.now();
        beds.forEach((bed) => {
          bed.deviceOnline = Boolean(bed.latestTimestamp && now - bed.latestTimestamp <= deviceOfflineMs);
        });
      };
  
      const applyOverview = (overview) => {
        const apiBeds = new Map();
        (overview.rooms || []).forEach((room) => {
          (room.beds || []).forEach((bed) => apiBeds.set(bed.bed_id, bed));
        });
  
        beds.forEach((bed, idx) => {
          const incoming = apiBeds.get(bed.id);
          if (!incoming) return;
          const previousDataId = bed.latestDataId ?? null;
          const incomingDataId = incoming.latest?.data_id ?? null;
          bed.sensorNumber = incoming.sensor_number || bed.sensorNumber;
          bed.deviceType = incoming.device_type || bed.deviceType;
          bed.patient = incoming.patient_name || bed.patient;
          bed.latestDataId = incomingDataId;
          applyPointToBed(bed, incoming.latest, { shouldPushHistory: !hasFixedRange(), updateLiveness: true });
          bed.deviceOnline = Boolean(incoming.device_online);
          if (incoming.presence !== undefined) bed.presence = Boolean(incoming.presence);
          updateBedDom(bed, idx, incomingDataId !== null && incomingDataId !== previousDataId);
        });
        updateSummary();
        renderThermalPanel();
        renderSelectedCharts();
      };

      const applyRealtimeEvent = (event) => {
        const latest = event?.latest;
        const sensorNumber = event?.sensor_number || latest?.sensor_number;
        if (!sensorNumber || !latest) return;
        const idx = beds.findIndex((bed) => bed.sensorNumber === sensorNumber);
        if (idx < 0) return;
        const bed = beds[idx];
        const previousDataId = bed.latestDataId ?? null;
        bed.deviceType = event.device_type || bed.deviceType;
        bed.deviceOnline = Boolean(event.device_online);
        if (event.presence !== undefined) bed.presence = Boolean(event.presence);
        bed.latestDataId = latest.data_id ?? bed.latestDataId;
        applyPointToBed(bed, latest, { shouldPushHistory: !hasFixedRange(), updateLiveness: true });
        updateBedDom(bed, idx, bed.latestDataId !== previousDataId);
        updateSummary();
        if (bed.id === selectedBedId) {
          renderThermalPanel();
          renderSelectedCharts();
        }
      };

      const stopWardStream = () => {
        if (wardStream) {
          wardStream.close();
          wardStream = null;
        }
        streamConnected = false;
      };

      const startWardStream = () => {
        stopWardStream();
        try {
          wardStream = openWardFloorStream(selectedFloor);
          wardStream.addEventListener('ready', () => {
            apiMode = true;
            streamConnected = true;
            fetchOverview();
          });
          wardStream.addEventListener('sensor_data', (event) => {
            apiMode = true;
            streamConnected = true;
            try {
              applyRealtimeEvent(JSON.parse(event.data));
            } catch (err) {
              console.warn('ward stream event parse failed', err);
              fetchOverview();
            }
          });
          wardStream.onerror = () => {
            streamConnected = false;
            fetchOverview();
          };
        } catch (err) {
          streamConnected = false;
          console.warn('ward stream unavailable, fallback to polling', err);
        }
      };
  
    const fetchOverview = async () => {
      lastOverviewFetchMs = Date.now();
      try {
        const data = await fetchWardFloorOverview(selectedFloor);
        apiMode = true;
        applyOverview(data);
      } catch (err) {
        apiMode = false;
        console.warn('ward overview api unavailable, fallback to mock data', err);
        }
      };
  
      const buildHistoryQuery = () => {
        const params = new URLSearchParams();
        if (rangeStartMs === null && rangeEndMs === null) {
          params.set('start', new Date(Date.now() - liveChartWindowMs).toISOString());
        } else if (rangeStartMs !== null) {
          params.set('start', new Date(rangeStartMs).toISOString());
        }
        if (rangeEndMs !== null) params.set('end', new Date(rangeEndMs).toISOString());
        return params.toString();
      };
  
      const refreshSelectedHistory = async (options = {}) => {
        const { force = false } = options;
        if (!selectedBedId || !apiMode || (selectedHistoryInFlight && !force)) return;
        const bed = beds.find((b) => b.id === selectedBedId);
      if (!bed?.sensorNumber) return;
      selectedHistoryInFlight = true;
      lastSelectedHistoryFetchMs = Date.now();
      const token = ++historyRequestToken;
      try {
        const query = buildHistoryQuery();
        const rangeActive = rangeStartMs !== null || rangeEndMs !== null;
        const [data, thermalTimeline] = await Promise.all([
          fetchSensorHistory(bed.sensorNumber, query),
          rangeActive ? fetchSensorThermalTimeline(bed.sensorNumber, query) : Promise.resolve(null),
        ]);
        if (token !== historyRequestToken) return;
        resetHistory(bed);
        setHistoryFromPoints(bed, data.points || []);
        if (rangeActive) {
          setThermalTimeline(bed, thermalTimeline?.frames || []);
          thermalTimelineIndex = bed.thermalHistory.length > 0
            ? clampInt(thermalTimelineIndex ?? bed.thermalHistory.length - 1, 0, bed.thermalHistory.length - 1)
            : null;
        }
        renderThermalPanel();
        renderSelectedCharts();
      } catch (err) {
          console.warn('sensor history api unavailable, keep current history', err);
        } finally {
          if (token === historyRequestToken) selectedHistoryInFlight = false;
        }
      };

      const refreshSelectedThermalLatest = async (options = {}) => {
        const { force = false } = options;
        if (!props.showThermal || hasFixedRange() || !selectedBedId || !apiMode || (selectedThermalInFlight && !force)) return;
        const bed = beds.find((b) => b.id === selectedBedId);
        if (!bed?.sensorNumber || bed.presence === false || !bed.deviceOnline) return;
        selectedThermalInFlight = true;
        lastSelectedThermalFetchMs = Date.now();
        try {
          const point = await fetchSensorThermalLatest(bed.sensorNumber);
          const frame = parseTemperatureFrame(point?.temperature_json);
          if (!frame) return;
          const ts = new Date(point.timestamp).getTime();
          bed.thermalFrame = frame;
          if (Number.isFinite(ts)) {
            pushThermalHistoryFrame(bed, ts, frame);
          }
          renderThermalPanel();
        } catch (err) {
          console.warn('sensor latest thermal api unavailable', err);
        } finally {
          selectedThermalInFlight = false;
        }
      };
  
      const seedHistory = (bed) => {
        const now = Date.now();
        // 回填 7 天資料：遠期用較粗粒度、近期用較細粒度，兼顧趨勢與即時感
        const oldDays = 7;
        const oldStepMs = 15 * 60 * 1000; // 15 分鐘
        const recentWindowMs = 2 * 60 * 60 * 1000; // 最近 2 小時
        const recentStepMs = 30 * 1000; // 30 秒
  
        const oldStart = now - oldDays * 24 * 60 * 60 * 1000;
        const oldEnd = now - recentWindowMs;
        for (let ts = oldStart; ts < oldEnd; ts += oldStepMs) {
          bed.temperature = Number(clamp(bed.temperature + (Math.random() - 0.5) * 0.16, 35.0, 39.5).toFixed(1));
          bed.heartRate = clamp(Math.round(bed.heartRate + (Math.random() - 0.5) * 4), 38, 145);
          bed.breath = clamp(Math.round(bed.breath + (Math.random() - 0.5) * 3), 8, 32);
          bed.rhythm = '竇性心律';
          pushHistoryPoint(bed, ts);
        }
  
        for (let ts = oldEnd; ts <= now; ts += recentStepMs) {
          bed.temperature = Number(clamp(bed.temperature + (Math.random() - 0.5) * 0.2, 35.0, 39.5).toFixed(1));
          bed.heartRate = clamp(Math.round(bed.heartRate + (Math.random() - 0.5) * 5), 38, 145);
          bed.breath = clamp(Math.round(bed.breath + (Math.random() - 0.5) * 3), 8, 32);
          bed.rhythm = '竇性心律';
          pushHistoryPoint(bed, ts);
        }
      };
  
      function sortedBedPoints(room) {
        const override = roomGridOverrides[room.id] || baseRoomGridOverrides[room.base] || {};
        const cols = override.cols ?? (room.beds <= 2 ? 1 : 2);
        const rows = override.rows ?? Math.ceil(room.beds / cols);
        const padX = override.padX ?? 7;
        const padY = override.padY ?? 10;
        const areaW = room.w - padX * 2;
        const areaH = room.h - padY * 2;
        // 統一床長度
        const bedW = 73;
        const bedH = Math.min(40, areaH / rows - 4);
  
        const pts = [];
        for (let r = 0; r < rows; r += 1) {
          for (let c = 0; c < cols; c += 1) {
            const cx = room.x + padX + (c + 0.5) * (areaW / cols);
            const cy = room.y + padY + (r + 0.5) * (areaH / rows);
            pts.push({ cx, cy, bedW, bedH });
          }
        }

        pts.sort((a, b) => {
          const doorX = (room.door.x1 + room.door.x2) / 2;
          const doorY = (room.door.y1 + room.door.y2) / 2;
          const da = Math.hypot(a.cx - doorX, a.cy - doorY);
          const db = Math.hypot(b.cx - doorX, b.cy - doorY);
          if (Math.abs(da - db) > 0.01) return da - db;
          if (Math.abs(a.cy - b.cy) > 0.01) return a.cy - b.cy;
          return a.cx - b.cx;
        });
  
        return pts.slice(0, room.beds);
      }
  
      function renderBeds() {
        roomDefs.forEach((room, ri) => {
          const points = sortedBedPoints(room);
  
          points.forEach((p, bi) => {
            const mappedNo = customBedNumberMap[room.id]?.[bi] ?? baseBedNumberMap[room.base]?.[bi] ?? (bi + 1);
            const id = `${room.id}-${String(mappedNo).padStart(2, '0')}`;
            const baseHr = 62 + Math.floor(seeded(ri * 17 + bi + 1) * 35);
            const baseTemp = 36.2 + seeded(ri * 29 + bi + 2) * 0.9;
  
            const bed = {
              id,
              roomId: room.id,
              sensorNumber: `${room.id}_${String(mappedNo).padStart(2, '0')}`,
              deviceType: seeded(ri * 19 + bi + 3) > 0.5 ? 'esp32' : 'stm32',
              patient: randomName(ri * 100 + bi),
              rhythm: '竇性心律',
              temperature: Number(baseTemp.toFixed(1)),
              heartRate: baseHr,
              deviceOnline: seeded(ri * 31 + bi + 7) > 0.08,
              breath: 14 + Math.floor(seeded(ri * 23 + bi + 4) * 8),
              presence: true,
              latestDataId: null,
              latestTimestamp: null,
              lastUpdatedAt: null,
              thermalFrame: buildMockThermalFrame(baseTemp, ri * 1000 + bi * 37),
              thermalHistory: [],
              history: {
                ts: [],
                temp: [],
                hr: [],
                breath: []
              }
            };
            seedHistory(bed);
            beds.push(bed);
  
            const g = createSvg('g', {
              class: 'bed',
              'data-bed-id': id,
              transform: `translate(${p.cx - p.bedW / 2}, ${p.cy - p.bedH / 2})`
            });
  
            const rect = createSvg('rect', { width: p.bedW, height: p.bedH });
            const t1 = createSvg('text', { x: 3, y: 11, 'font-weight': 800, 'font-size': '10px' });
            const t2 = createSvg('text', { x: 3, y: 23, 'font-size': '10px', class: 'bed-patient' });
            
            t1.textContent = bed.sensorNumber;
            t2.textContent = bed.patient;
            
            const lightCount = 4;
            const lightGap = 14;
            const lightStartX = (p.bedW - (lightCount - 1) * lightGap) / 2;
            // 4 vital-sign indicators: heart rate, rhythm, temperature, breath.
            for (let i = 0; i < lightCount; i++) {
              const circle = createSvg('circle', {
                cx: lightStartX + i * lightGap,
                cy: 33,
                r: 4,
                class: 'light',
                'data-light-index': i
              });
              g.appendChild(circle);
            }
  
            g.prepend(rect);
            g.append(t1, t2);
            g.addEventListener('click', (e) => {
              e.stopPropagation();
              selectedRoomId = room.id;
              selectedBedId = id;
              thermalTimelineIndex = null;
              syncSelection();
              refreshSelectedHistory({ force: true });
            });
  
            bedsLayer.appendChild(g);
          });
        });
      }
  
      function updateBed(bed, idx) {
        if (Math.random() < 0.012) bed.deviceOnline = !bed.deviceOnline;
        if (Math.random() < 0.03) bed.presence = !bed.presence;
  
        const r = Math.random();
        if (r < 0.87) bed.rhythm = '竇性心律';
        else if (r < 0.94) bed.rhythm = '心房顫動';
        else if (r < 0.97) bed.rhythm = '心搏過速';
        else bed.rhythm = '心搏過緩';
  
        bed.heartRate = clamp(Math.round(bed.heartRate + (Math.random() - 0.5) * 8), 38, 145);
        bed.temperature = Number(clamp(bed.temperature + (Math.random() - 0.5) * 0.25, 35.0, 39.5).toFixed(1));
  
        if (bed.rhythm === '心搏過速') bed.heartRate = Math.max(112, bed.heartRate + 8);
        if (bed.rhythm === '心搏過緩') bed.heartRate = Math.min(49, bed.heartRate - 8);

        bed.breath = clamp(Math.round(bed.breath + (Math.random() - 0.5) * 3), 8, 32);
        bed.thermalFrame = buildMockThermalFrame(
          bed.temperature,
          Number((bed.latestTimestamp || Date.now()) + idx * 17),
        );
        const ts = Date.now();
        pushHistoryPoint(bed, ts);
        pushThermalHistoryFrame(bed, ts, bed.thermalFrame);
        updateBedDom(bed, idx);
      }
  
      function updateBedDom(bed, idx, markUpdated = false) {
        const g = bedsLayer.children[idx];
        if (!g) return;

        const isHrOk = bed.heartRate >= 50 && bed.heartRate <= 110;
        const isRhythmOk = bed.rhythm === '竇性心律';
        const isTempOk = bed.temperature >= 35.8 && bed.temperature <= 37.6;
        const isBreathOk = bed.breath >= 10 && bed.breath <= 25;
        const isInBed = bed.presence !== false;
        
        const lights = g.querySelectorAll('circle.light');
        if (lights.length === 4) {
          if (!bed.deviceOnline) {
            lights.forEach((light) => light.setAttribute('class', 'light hidden'));
          } else if (!isInBed) {
            lights.forEach((light) => light.setAttribute('class', 'light disabled'));
          } else {
            lights[0].setAttribute('class', `light ${isHrOk ? 'hr-ok' : 'hr-warn'}`);
            lights[1].setAttribute('class', `light ${isRhythmOk ? 'rhythm-ok' : 'rhythm-warn'}`);
            lights[2].setAttribute('class', `light ${isTempOk ? 'temp-ok' : 'temp-warn'}`);
            lights[3].setAttribute('class', `light ${isBreathOk ? 'breath-ok' : 'breath-warn'}`);
          }
        }

        const hasAlert = bed.deviceOnline && isInBed && hasWarningLight(bed);
        
        g.classList.toggle('alert', hasAlert);
        g.classList.toggle('offline', !bed.deviceOnline);
        g.classList.toggle('out-of-bed', false);
        g.classList.toggle('selected', selectedBedId === bed.id);

        const patient = g.querySelector('.bed-patient');
        if (patient) patient.textContent = bed.patient;

        if (markUpdated) {
          g.classList.add('updated');
          if (updatedClassTimers.has(bed.id)) clearTimeout(updatedClassTimers.get(bed.id));
          updatedClassTimers.set(bed.id, setTimeout(() => {
            g.classList.remove('updated');
            updatedClassTimers.delete(bed.id);
          }, 420));
        }
      }
  
      function updateSummary() {
        const arr = beds;
        const online = arr.filter((b) => b.deviceOnline).length;
        const offline = arr.length - online;
        const outOfBed = 0;
        const alerts = arr.filter(hasWarningLight).length;
        document.getElementById('summary').textContent = t('ward.summary.text', {
          online,
          offline,
          outOfBed,
          alerts,
        });
      }
  
      function getFilteredHistory(bed) {
        const result = { ts: [], temp: [], hr: [], breath: [] };
        for (let i = 0; i < bed.history.ts.length; i += 1) {
          const ts = bed.history.ts[i];
          if ((rangeStartMs === null || ts >= rangeStartMs) && (rangeEndMs === null || ts <= rangeEndMs)) {
            result.ts.push(ts);
            result.temp.push(bed.history.temp[i]);
            result.hr.push(bed.history.hr[i]);
            result.breath.push(bed.history.breath[i]);
          }
        }
        return result;
      }
  
      function clearChart(svgId) {
        const svg = document.getElementById(svgId);
        if (svg) svg.innerHTML = '';
      }
  
      function drawNoData(svgId, msg) {
        const svg = document.getElementById(svgId);
        if (!svg) return;
        svg.appendChild(createSvg('text', {
          x: chartBox.w / 2,
          y: chartBox.h / 2 + 2,
          'text-anchor': 'middle',
          'font-size': 12,
          fill: '#83a9cc'
        }));
        svg.lastChild.textContent = msg;
      }
  
      const chartBox = { w: 300, h: 126, pLeft: 30, pRight: 8, pTop: 10, pBottom: 26 };
  
      const formatTimeTick = (ts, spanMs) => {
        const d = new Date(ts);
        const hhmm = `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
        if (spanMs >= 365 * 24 * 60 * 60 * 1000) return [`${d.getFullYear()}/${pad2(d.getMonth() + 1)}`];
        if (spanMs >= 24 * 60 * 60 * 1000) return [`${pad2(d.getMonth() + 1)}/${pad2(d.getDate())}`, hhmm];
        return [hhmm];
      };
  
      const pickTickSegments = (spanMs) => {
        const hour = 60 * 60 * 1000;
        const day = 24 * hour;
        if (spanMs >= 365 * day) return 3; // 4 個刻度
        if (spanMs >= 180 * day) return 4; // 5 個刻度
        if (spanMs >= 90 * day) return 3;  // 4 個刻度
        if (spanMs >= 30 * day) return 2;  // 3 個刻度（例如 1/15/30）
        if (spanMs >= 14 * day) return 3;
        if (spanMs >= 7 * day) return 4;
        if (spanMs >= day) return 5;
        if (spanMs >= 6 * hour) return 5;
        return 4;
      };
  
      const buildTimeTicks = (xDomain) => {
        const spanMs = Math.max(1, xDomain.end - xDomain.start);
        const segments = pickTickSegments(spanMs);
        const ticks = [];
        for (let i = 0; i <= segments; i += 1) {
          ticks.push(Math.round(xDomain.start + (spanMs * i) / segments));
        }
        return ticks;
      };
  
      const getXDomain = (tsValues) => {
        const dataStart = tsValues.length ? tsValues[0] : null;
        const dataEnd = tsValues.length ? tsValues[tsValues.length - 1] : null;

        if (rangeStartMs === null && rangeEndMs === null) {
          const end = Math.max(Date.now(), dataEnd || Date.now());
          return { start: end - liveChartWindowMs, end };
        }
  
        let start = rangeStartMs !== null ? rangeStartMs : dataStart;
        let end = rangeEndMs !== null ? rangeEndMs : dataEnd;
  
        if (start === null && end === null) {
          const now = Date.now();
          start = now - 60 * 1000;
          end = now;
        } else if (start === null) {
          start = end - 60 * 1000;
        } else if (end === null) {
          end = start + 60 * 1000;
        }
  
        if (end <= start) end = start + 60 * 1000;
        return { start, end };
      };
  
      function drawAxesAndGrid(svg, xDomain, yMin, yMax, yTickCount, yFormatter) {
        const { w, h, pLeft, pRight, pTop, pBottom } = chartBox;
        const plotW = w - pLeft - pRight;
        const plotH = h - pTop - pBottom;
        const ySpan = Math.max(1, yMax - yMin);
  
        const yToPx = (value) => pTop + (1 - (clamp(value, yMin, yMax) - yMin) / ySpan) * plotH;
  
        for (let i = 0; i <= yTickCount; i += 1) {
          const ratio = i / yTickCount;
          const val = yMax - ratio * (yMax - yMin);
          const y = yToPx(val);
          svg.appendChild(createSvg('line', {
            x1: pLeft,
            y1: y,
            x2: w - pRight,
            y2: y,
            stroke: 'rgba(162, 201, 235, 0.28)',
            'stroke-width': 1
          }));
          svg.appendChild(createSvg('line', {
            x1: pLeft - 3,
            y1: y,
            x2: pLeft,
            y2: y,
            stroke: '#7eb4e4',
            'stroke-width': 1
          }));
          const yText = createSvg('text', {
            x: pLeft - 5,
            y,
            'text-anchor': 'end',
            'dominant-baseline': 'middle',
            'font-size': 7,
            fill: '#9fd2ff'
          });
          yText.textContent = yFormatter(val);
          svg.appendChild(yText);
        }
  
        const spanMs = Math.max(1, xDomain.end - xDomain.start);
        const timeTicks = buildTimeTicks(xDomain);
        timeTicks.forEach((tickTs, tickIndex) => {
          const ratio = clamp((tickTs - xDomain.start) / spanMs, 0, 1);
          const x = pLeft + ratio * plotW;
          svg.appendChild(createSvg('line', {
            x1: x,
            y1: pTop,
            x2: x,
            y2: h - pBottom,
            stroke: 'rgba(162, 201, 235, 0.20)',
            'stroke-width': 1
          }));
          svg.appendChild(createSvg('line', {
            x1: x,
            y1: h - pBottom,
            x2: x,
            y2: h - pBottom + 3,
            stroke: '#7eb4e4',
            'stroke-width': 1
          }));
          const textAnchor = tickIndex === 0 ? 'start' : tickIndex === timeTicks.length - 1 ? 'end' : 'middle';
          const labelX = tickIndex === 0 ? x + 1 : tickIndex === timeTicks.length - 1 ? x - 1 : x;
          const xText = createSvg('text', {
            x: labelX,
            y: formatTimeTick(tickTs, spanMs).length > 1 ? h - 13 : h - 4,
            'text-anchor': textAnchor,
            'font-size': spanMs >= 24 * 60 * 60 * 1000 ? 7 : 8,
            fill: '#9fd2ff'
          });
          formatTimeTick(tickTs, spanMs).forEach((line, lineIndex) => {
            const tspan = createSvg('tspan', {
              x: labelX,
              dy: lineIndex === 0 ? 0 : 8,
            });
            tspan.textContent = line;
            xText.appendChild(tspan);
          });
          svg.appendChild(xText);
        });
  
        svg.appendChild(createSvg('line', {
          x1: pLeft,
          y1: pTop,
          x2: pLeft,
          y2: h - pBottom,
          stroke: '#7eb4e4',
          'stroke-width': 1.2
        }));
        svg.appendChild(createSvg('line', {
          x1: pLeft,
          y1: h - pBottom,
          x2: w - pRight,
          y2: h - pBottom,
          stroke: '#7eb4e4',
          'stroke-width': 1.2
        }));
      }
  
      function pointsToPath(points) {
        if (!points.length) return '';
        if (points.length === 1) {
          const p = points[0];
          return `M ${p.x} ${p.y} L ${p.x} ${p.y}`;
        }
        return points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
      }

      function valueDomain(values, fallbackMin, fallbackMax, paddingRatio = 0.12) {
        const finite = values.filter((value) => Number.isFinite(value));
        if (!finite.length) return { min: fallbackMin, max: fallbackMax };
        let min = Math.min(...finite);
        let max = Math.max(...finite);
        if (min === max) {
          min -= 1;
          max += 1;
        }
        const padding = Math.max((max - min) * paddingRatio, 0.5);
        return { min: min - padding, max: max + padding };
      }

      function downsampleSeriesForChart(tsValues, values, xDomain, maxBuckets = 900) {
        if (values.length <= maxBuckets) return { ts: tsValues, values };
        const bucketCount = Math.max(1, maxBuckets);
        const span = Math.max(1, xDomain.end - xDomain.start);
        const buckets = Array.from({ length: bucketCount }, () => null);

        for (let i = 0; i < values.length; i += 1) {
          const value = values[i];
          const ts = tsValues[i];
          if (!Number.isFinite(value) || !Number.isFinite(ts)) continue;
          const bucketIndex = clampInt(Math.floor(((ts - xDomain.start) / span) * bucketCount), 0, bucketCount - 1);
          const bucket = buckets[bucketIndex] || {
            firstTs: ts,
            firstValue: value,
            minTs: ts,
            minValue: value,
            maxTs: ts,
            maxValue: value,
            lastTs: ts,
            lastValue: value,
          };
          if (value < bucket.minValue) {
            bucket.minValue = value;
            bucket.minTs = ts;
          }
          if (value > bucket.maxValue) {
            bucket.maxValue = value;
            bucket.maxTs = ts;
          }
          bucket.lastTs = ts;
          bucket.lastValue = value;
          buckets[bucketIndex] = bucket;
        }

        const sampled = [];
        buckets.forEach((bucket) => {
          if (!bucket) return;
          sampled.push({ ts: bucket.firstTs, value: bucket.firstValue });
          if (bucket.minTs <= bucket.maxTs) {
            sampled.push({ ts: bucket.minTs, value: bucket.minValue });
            sampled.push({ ts: bucket.maxTs, value: bucket.maxValue });
          } else {
            sampled.push({ ts: bucket.maxTs, value: bucket.maxValue });
            sampled.push({ ts: bucket.minTs, value: bucket.minValue });
          }
          sampled.push({ ts: bucket.lastTs, value: bucket.lastValue });
        });
        sampled.sort((a, b) => a.ts - b.ts);
        return {
          ts: sampled.map((point) => point.ts),
          values: sampled.map((point) => point.value),
        };
      }
  
      function seriesPoints(tsValues, values, min, max, xDomain) {
        const { w, h, pLeft, pRight, pTop, pBottom } = chartBox;
        const ySpan = Math.max(1, max - min);
        const xSpan = Math.max(1, xDomain.end - xDomain.start);
        const plotW = w - pLeft - pRight;
        const plotH = h - pTop - pBottom;
        return values.map((value, idx) => {
          const ts = tsValues[idx];
          const ratioX = tsValues.length ? clamp((ts - xDomain.start) / xSpan, 0, 1) : (values.length <= 1 ? 0 : idx / (values.length - 1));
          const x = pLeft + ratioX * plotW;
          const y = pTop + (1 - (clamp(value, min, max) - min) / ySpan) * plotH;
          return { x, y };
        });
      }
  
      function renderSingleLineChart(svgId, tsValues, values, min, max, color, yFormatter, xDomain) {
        const svg = document.getElementById(svgId);
        if (!svg) return;
        svg.innerHTML = '';
        drawAxesAndGrid(svg, xDomain, min, max, 4, yFormatter);
        if (!values.length) {
          drawNoData(svgId, t('ward.chart.noData'));
          return;
        }
  
        const sampled = downsampleSeriesForChart(tsValues, values, xDomain);
        const points = seriesPoints(sampled.ts, sampled.values, min, max, xDomain);
        const path = createSvg('path', {
          d: pointsToPath(points),
          fill: 'none',
          stroke: color,
          'stroke-width': 2.2,
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round'
        });
        svg.appendChild(path);
  
        const last = points[points.length - 1];
        svg.appendChild(createSvg('circle', { cx: last.x, cy: last.y, r: 2.8, fill: color }));
      }

      function validChartSeries(tsValues, values, isValid) {
        const ts = [];
        const cleanValues = [];
        for (let i = 0; i < values.length; i += 1) {
          const value = values[i];
          const timestamp = tsValues[i];
          if (!Number.isFinite(timestamp) || !Number.isFinite(value) || !isValid(value)) continue;
          ts.push(timestamp);
          cleanValues.push(value);
        }
        return { ts, values: cleanValues };
      }
  
      function resetChartNow() {
        document.getElementById('temp-now').textContent = '--';
        document.getElementById('hr-now').textContent = '--';
        document.getElementById('breath-now').textContent = '--';
      }
  
      function renderSelectedCharts() {
        if (!selectedBedId) {
          clearChart('temp-chart');
          clearChart('hr-chart');
          clearChart('breath-chart');
          drawNoData('temp-chart', t('ward.selection.pickBedFirst'));
          drawNoData('hr-chart', t('ward.selection.pickBedFirst'));
          drawNoData('breath-chart', t('ward.selection.pickBedFirst'));
          resetChartNow();
          return;
        }
  
        const bed = beds.find((b) => b.id === selectedBedId);
        if (!bed) {
          resetChartNow();
          return;
        }
        const history = getFilteredHistory(bed);
        const tempSeries = validChartSeries(history.ts, history.temp, (value) => value > 0);
        const hrSeries = validChartSeries(history.ts, history.hr, (value) => value > 0);
        const breathSeries = validChartSeries(history.ts, history.breath, (value) => value > 0);
        const xDomain = getXDomain(history.ts);
        const tempDomain = valueDomain(tempSeries.values, 29.0, 40.0);
        const hrDomain = valueDomain(hrSeries.values, 40, 150);
        const breathDomain = valueDomain(breathSeries.values, 8, 32);
        renderSingleLineChart('temp-chart', tempSeries.ts, tempSeries.values, tempDomain.min, tempDomain.max, '#7dd3fc', (v) => v.toFixed(1), xDomain);
        renderSingleLineChart('hr-chart', hrSeries.ts, hrSeries.values, Math.min(40, hrDomain.min), Math.max(150, hrDomain.max), '#60a5fa', (v) => String(Math.round(v)), xDomain);
        renderSingleLineChart('breath-chart', breathSeries.ts, breathSeries.values, Math.min(8, breathDomain.min), Math.max(32, breathDomain.max), '#34d399', (v) => String(Math.round(v)), xDomain);
        if (bed.deviceOnline && bed.presence !== false && history.ts.length > 0) {
          const i = history.ts.length - 1;
          const latestTemp = history.temp[i];
          const latestHr = history.hr[i];
          const latestBreath = history.breath[i];
          document.getElementById('temp-now').textContent = Number.isFinite(latestTemp) ? `${latestTemp.toFixed(1)} ${t('ward.units.temperature')}` : '--';
          document.getElementById('hr-now').textContent = Number.isFinite(latestHr) && latestHr > 0 ? `${latestHr} ${t('ward.units.heartRate')}` : '--';
          document.getElementById('breath-now').textContent = Number.isFinite(latestBreath) && latestBreath > 0 ? `${latestBreath} ${t('ward.units.breath')}` : '--';
        } else {
          resetChartNow();
        }
      }
  
      function syncSelection() {
        document.querySelectorAll('.room-shape').forEach((shape) => {
          const id = shape.parentNode.id.replace('room-', '');
          shape.classList.toggle('active', id === selectedRoomId);
        });
  
        Array.from(bedsLayer.children).forEach((g, i) => {
          g.classList.toggle('selected', beds[i]?.id === selectedBedId);
        });
  
        const title = document.getElementById('sel-title');
        const meta = document.getElementById('sel-meta');
  
        if (!selectedBedId) {
          if (selectedRoomId) {
            title.textContent = t('ward.selection.roomTitle', { room: selectedRoomId });
            meta.innerHTML = `
              <div class="meta-col">
                <div class="kv"><b>${t('ward.fields.status')}</b><span>${t('ward.selection.roomSelected')}</span></div>
                <div class="kv"><b>${t('ward.fields.rhythm')}</b><span>-</span></div>
                <div class="kv"><b>${t('ward.fields.temperature')}</b><span>-</span></div>
              </div>
              <div class="meta-col">
                <div class="kv"><b>${t('ward.fields.heartRate')}</b><span>-</span></div>
                <div class="kv"><b>${t('ward.fields.breath')}</b><span>-</span></div>
                <div class="kv"><b>${t('ward.fields.device')}</b><span>-</span></div>
              </div>
            `;
          } else {
            title.textContent = t('ward.selection.noBed');
            meta.innerHTML = `
              <div class="meta-col">
                <div class="kv"><b>${t('ward.fields.patient')}</b><span>${t('ward.selection.pickAnyBed')}</span></div>
                <div class="kv"><b>${t('ward.fields.rhythm')}</b><span>-</span></div>
                <div class="kv"><b>${t('ward.fields.temperature')}</b><span>-</span></div>
              </div>
              <div class="meta-col">
                <div class="kv"><b>${t('ward.fields.heartRate')}</b><span>-</span></div>
                <div class="kv"><b>${t('ward.fields.breath')}</b><span>-</span></div>
                <div class="kv"><b>${t('ward.fields.device')}</b><span>-</span></div>
              </div>
            `;
          }
          updateSummary();
          renderThermalPanel();
          renderSelectedCharts();
          return;
        }
  
        const bed = beds.find((b) => b.id === selectedBedId);
        if (!bed) return;
        const isCurrentDetecting = bed.deviceOnline && bed.presence !== false;
        const rhythmValue = isCurrentDetecting ? rhythmText(bed.rhythm) : '-';
        const temperatureValue = isCurrentDetecting ? `${bed.temperature.toFixed(1)} ${t('ward.units.temperature')}` : '-';
        const heartRateValue = isCurrentDetecting ? `${bed.heartRate} ${t('ward.units.heartRate')}` : '-';
        const breathValue = isCurrentDetecting ? `${bed.breath} ${t('ward.units.breath')}` : '-';
  
        title.textContent = t('ward.selection.bedTitle', { bed: bed.id });
        meta.innerHTML = `
          <div class="meta-col">
            <div class="kv"><b>${t('ward.fields.patient')}</b><span>${bed.patient}</span></div>
            <div class="kv"><b>${t('ward.fields.rhythm')}</b><span>${rhythmValue}</span></div>
            <div class="kv"><b>${t('ward.fields.temperature')}</b><span>${temperatureValue}</span></div>
          </div>
          <div class="meta-col">
            <div class="kv"><b>${t('ward.fields.heartRate')}</b><span>${heartRateValue}</span></div>
            <div class="kv"><b>${t('ward.fields.breath')}</b><span>${breathValue}</span></div>
            <div class="kv"><b>${t('ward.fields.device')}</b><span>${deviceStatusText(bed.deviceOnline)}</span></div>
            <div class="kv"><b>${t('ward.fields.presence')}</b><span>${presenceStatusText(bed.presence !== false)}</span></div>
          </div>
        `;
        updateSummary();
        renderThermalPanel();
        renderSelectedCharts();
      }
  
      function tick() {
        const now = Date.now();
        if (apiMode) {
          refreshDeviceOnlineStates();
          beds.forEach((b, i) => updateBedDom(b, i));
          if (now - lastOverviewFetchMs >= overviewPollMs) fetchOverview();
          if (!hasFixedRange() && selectedBedId && now - lastSelectedHistoryFetchMs >= selectedHistoryPollMs) refreshSelectedHistory();
          if (selectedBedId && now - lastSelectedThermalFetchMs >= selectedThermalPollMs) refreshSelectedThermalLatest();
        } else {
          beds.forEach((b, i) => updateBed(b, i));
        }
        document.getElementById('clock').textContent = new Date().toLocaleString(selectedLanguage, {
          hour12: false,
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit'
        });
        syncSelection();
      }
  
      document.getElementById('plan').addEventListener('click', () => {
        selectedBedId = null;
        syncSelection();
      });
  
      document.getElementById('apply-range').addEventListener('click', () => {
        let start = parsePickerMs(rangeStartPicker.value);
        let end = parsePickerMs(rangeEndPicker.value);
        if (start !== null && end !== null && start > end) {
          const tmp = start;
          start = end;
          end = tmp;
        }
        rangeStartMs = start;
        rangeEndMs = end;
        thermalTimelineIndex = null;
        updateRangeInputs();
        renderThermalPanel();
        renderSelectedCharts();
        refreshSelectedHistory({ force: true });
      });
  
      const applyQuickRange = (durationMs) => {
        rangeEndMs = Date.now();
        rangeStartMs = rangeEndMs - durationMs;
        thermalTimelineIndex = null;
        updateRangeInputs();
        renderThermalPanel();
        renderSelectedCharts();
        refreshSelectedHistory({ force: true });
      };
  
      document.getElementById('quick-5m').addEventListener('click', () => applyQuickRange(5 * 60 * 1000));
      document.getElementById('quick-1h').addEventListener('click', () => applyQuickRange(60 * 60 * 1000));
      document.getElementById('quick-1d').addEventListener('click', () => applyQuickRange(24 * 60 * 60 * 1000));
      document.getElementById('quick-3d').addEventListener('click', () => applyQuickRange(3 * 24 * 60 * 60 * 1000));
      document.getElementById('quick-7d').addEventListener('click', () => applyQuickRange(7 * 24 * 60 * 60 * 1000));
  
      document.getElementById('clear-range').addEventListener('click', () => {
        rangeStartMs = null;
        rangeEndMs = null;
        thermalTimelineIndex = null;
        updateRangeInputs();
        renderThermalPanel();
        renderSelectedCharts();
        refreshSelectedHistory({ force: true });
      });

      byId('thermal-history-slider')?.addEventListener('input', (event) => {
        thermalTimelineIndex = Number(event.target.value);
        renderThermalPanel();
      });

      byId('floor-select')?.addEventListener('click', (event) => {
        const option = event.target.closest('[data-floor]');
        if (!option) return;
        const nextFloor = Number(option.dataset.floor);
        switchFloor(nextFloor);
      });
      byId('floor-prev')?.addEventListener('click', () => stepFloor(-1));
      byId('floor-next')?.addEventListener('click', () => stepFloor(1));
      byId('language-select')?.addEventListener('click', (event) => {
        const option = event.target.closest('[data-lang]');
        if (!option) return;
        setLanguage(option.dataset.lang);
      });
      byId('language-prev')?.addEventListener('click', () => stepLanguage(-1));
      byId('language-next')?.addEventListener('click', () => stepLanguage(1));
  
      await loadFloorOptions();
      setLanguage(selectedLanguage);
      renderRooms();
      renderBeds();
      updateRangeInputs();
      fetchOverview();
      startWardStream();
      tick();
      timer = setInterval(tick, 500);
  cleanup = () => {
    if (timer) clearInterval(timer)
    updatedClassTimers.forEach((timeoutID) => clearTimeout(timeoutID))
    updatedClassTimers.clear()
    stopWardStream()
  }
})

onUnmounted(() => cleanup())
</script>

<style>
    :root {
      --bg: #eaf0f7;
      --panel: #f8fbff;
      --ink: #132844;
      --muted: #4c6784;
      --line: #2f5d86;
      --room: #f6fbff;
      --corridor: #dff0df;
      --bed: #ffffff;
      --out-of-bed: #f59e0b;
      --ok: #16a34a;
      --warn: #d97706;
      --danger: #be123c;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: radial-gradient(circle at 10% 10%, #f5faff 0, #eaf0f7 38%), linear-gradient(180deg, #edf3fa, #e4ebf4);
      color: var(--ink);
      font-family: "Noto Sans TC", "PingFang TC", sans-serif;
    }
    .layout {
      min-height: 100vh;
      display: grid;
      grid-template-columns: minmax(820px, 1fr) 420px;
      gap: 14px;
      padding: 14px;
    }
    .card {
      background: var(--panel);
      border: 1px solid #c8d8ea;
      border-radius: 16px;
      box-shadow: 0 12px 30px rgba(14, 35, 61, .12);
    }
    .left { padding: 12px; overflow: auto; }
    .head {
      display: flex; justify-content: space-between; align-items: center;
      gap: 10px; flex-wrap: nowrap; margin-bottom: 10px;
    }
    .head-main {
      min-width: 0;
      flex: 1 1 auto;
    }
    .head h1 {
      margin: 0;
      font-size: 18px;
      letter-spacing: .03em;
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: nowrap;
      min-width: 0;
    }
    #current-floor-title {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 50px;
      border-radius: 14px;
      padding: 6px 11px;
      background: linear-gradient(135deg, #10385f 0%, #2f8df5 100%);
      color: #fff;
      box-shadow: 0 8px 18px rgba(47, 141, 245, .24);
    }
    .ward-title-text {
      display: inline-flex;
      align-items: center;
      min-width: 210px;
      flex: 0 0 210px;
      white-space: nowrap;
    }
    .head.head-engineer #current-floor-title,
    .head.head-engineer .ward-title-text {
      transform: translateY(-4px);
    }
    .floor-inline {
      display: inline-flex;
      align-items: center;
      gap: 7px;
      min-width: 0;
      transform: translateY(1px);
    }
    .language-inline {
      display: inline-flex;
      align-items: center;
      gap: 7px;
      min-width: 0;
      transform: translateY(1px);
    }
    .status-inline {
      display: inline-flex;
      min-width: 0;
      flex: 0 1 auto;
      margin-left: 8px;
      max-width: 100%;
      overflow: hidden;
    }
    .floor-window {
      display: inline-flex;
      min-width: 0;
      width: 186px;
      max-width: 186px;
      overflow: hidden;
      border: 1px solid #b7cbe0;
      border-radius: 18px;
      background: rgba(255,255,255,.76);
      box-shadow: inset 0 1px 0 rgba(255,255,255,.96);
    }
    .language-window {
      display: inline-flex;
      min-width: 0;
      width: 186px;
      max-width: 186px;
      overflow: hidden;
      border: 1px solid #b7cbe0;
      border-radius: 18px;
      background: rgba(255,255,255,.76);
      box-shadow: inset 0 1px 0 rgba(255,255,255,.96);
    }
    .floor-switch {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      min-width: 0;
      width: 100%;
      overflow-x: auto;
      overscroll-behavior-x: contain;
      padding: 3px;
      scrollbar-width: none;
    }
    .floor-switch::-webkit-scrollbar {
      display: none;
    }
    .floor-nav,
    .language-nav {
      width: 34px;
      height: 34px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      border: 1px solid #a9c2dc;
      border-radius: 50%;
      background: #f4f8fc;
      color: #173d63;
      font-size: 24px;
      font-weight: 900;
      line-height: 1;
      padding: 0 0 3px;
      box-shadow: 0 6px 14px rgba(34, 77, 117, .12);
      transition: background .18s ease, border-color .18s ease, color .18s ease, transform .18s ease, opacity .18s ease;
    }
    .floor-nav:hover:not(:disabled),
    .language-nav:hover:not(:disabled) {
      background: #e7f1fb;
      border-color: #7ea8d3;
      transform: translateY(-1px);
    }
    .floor-nav:disabled,
    .language-nav:disabled {
      opacity: .38;
      cursor: not-allowed;
      box-shadow: none;
    }
    .floor-option {
      flex: 0 0 auto;
      min-width: 54px;
      border: 0;
      border-radius: 14px;
      background: transparent;
      color: #416381;
      font-size: 14px;
      font-weight: 900;
      letter-spacing: .03em;
      padding: 7px 12px;
      box-shadow: none;
      transition: background .18s ease, color .18s ease, transform .18s ease, box-shadow .18s ease;
    }
    .floor-option:hover {
      color: #12365d;
      transform: translateY(-1px);
    }
    .floor-option.active {
      background: linear-gradient(135deg, #12365d 0%, #2f8df5 100%);
      color: #fff;
      box-shadow: 0 7px 16px rgba(47, 141, 245, .28);
    }
    .head small { color: var(--muted); font-size: 12px; }
    .actions {
      display: flex;
      flex-direction: column;
      gap: 7px;
      align-items: flex-end;
      justify-content: flex-end;
      flex-wrap: nowrap;
      min-width: 280px;
    }
    .engineer-logout-row {
      position: relative;
      z-index: 3;
      height: 0;
      padding-left: 0;
    }
    .engineer-logout {
      position: relative;
      top: -22px;
      border: 1px solid #7f1d1d;
      border-radius: 999px;
      background: linear-gradient(135deg, #7f1d1d 0%, #b91c1c 55%, #dc2626 100%);
      color: #fff;
      font-size: 11px;
      font-weight: 800;
      letter-spacing: .02em;
      line-height: 1.15;
      padding: 3px 11px;
      box-shadow: 0 8px 16px rgba(127, 29, 29, .28);
      transition: transform .18s ease, box-shadow .18s ease, filter .18s ease;
    }
    .engineer-logout:hover {
      transform: translateY(-1px);
      filter: brightness(1.04);
      box-shadow: 0 10px 18px rgba(127, 29, 29, .34);
    }
    .engineer-logout:active {
      transform: translateY(0);
      box-shadow: 0 5px 12px rgba(127, 29, 29, .32);
    }
    .actions .floor-inline {
      margin-left: 0;
    }
    .actions .language-inline {
      margin-left: 0;
    }
    .language-switch {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 3px;
      min-width: 0;
      width: 100%;
      overflow-x: auto;
      overscroll-behavior-x: contain;
      scrollbar-width: none;
    }
    .language-switch::-webkit-scrollbar {
      display: none;
    }
    .language-option {
      min-width: 54px;
      flex: 0 0 auto;
      border: 0;
      border-radius: 14px;
      background: transparent;
      color: #416381;
      font-size: 14px;
      font-weight: 900;
      padding: 7px 12px;
      box-shadow: none;
      transition: background .18s ease, color .18s ease, transform .18s ease, box-shadow .18s ease;
    }
    .language-option:hover {
      color: #12365d;
      transform: translateY(-1px);
    }
    .language-option.active {
      background: #173d63;
      color: #fff;
      box-shadow: 0 7px 16px rgba(23, 61, 99, .22);
    }
    .legend-box {
      display: grid;
      grid-template-columns: repeat(4, 86px) 102px;
      gap: 6px;
      font-size: 11px;
      background: rgba(255,255,255,0.6);
      padding: 6px;
      border-radius: 12px;
      border: 1px solid #c6d7ea;
      box-shadow: inset 0 1px 2px rgba(0,0,0,0.05);
      width: max-content;
      max-width: 100%;
      min-width: 0;
    }
    .legend-item {
      display: grid;
      grid-template-columns: 1fr;
      gap: 4px;
      min-width: 0;
      padding: 7px 7px;
      border: 1px solid #d3e1ef;
      border-radius: 8px;
      background: rgba(255,255,255,.72);
      color: #1e3a8a;
      font-weight: 800;
      box-shadow: 0 1px 3px rgba(19, 49, 84, .06);
      overflow: hidden;
    }
    .legend-item small { font-weight: 400; color: #4b5563; }
    .legend-state { display: flex; align-items: center; gap: 4px; white-space: nowrap; }
    .legend-bed-state {
      gap: 3px;
      padding-top: 7px;
      padding-bottom: 6px;
    }
    .legend-bed-state .legend-state {
      gap: 3px;
    }
    .legend-bed-state small {
      font-size: 10px;
      line-height: 1;
    }
    html[lang="vi"] .legend-box {
      grid-template-columns: repeat(4, 84px) 100px;
    }
    html[lang="vi"] .legend-item {
      padding: 7px 8px;
    }
    html[lang="vi"] .legend-item small {
      font-size: 9px;
      line-height: 1.1;
    }
    html[lang="vi"] .legend-state {
      gap: 3px;
    }
    .dot { width: 9px; height: 9px; border-radius: 999px; display: inline-block; }
    .swatch {
      width: 12px;
      height: 9px;
      border-radius: 3px;
      display: inline-block;
      border: 1px solid rgba(29, 78, 216, .38);
    }
    .swatch.bed-online-in {
      background: #ffffff;
    }
    .swatch.bed-online-out {
      background: var(--out-of-bed);
      border-color: rgba(180, 83, 9, .55);
    }
    .swatch.bed-offline {
      background: #d1d5db;
      border-color: rgba(107, 114, 128, .5);
    }
    .dot.hr-ok,
    .dot.rhythm-ok,
    .dot.temp-ok,
    .dot.breath-ok {
      background: #16a34a;
      border: 1px solid rgba(21, 128, 61, .38);
    }
    .dot.hr-warn,
    .dot.rhythm-warn,
    .dot.temp-warn,
    .dot.breath-warn {
      background: #dc2626;
      border: 1px solid rgba(153, 27, 27, .38);
    }
    .light { stroke: rgba(0,0,0,0.05); stroke-width: 0.3; }
    .light.hr-ok,
    .light.rhythm-ok,
    .light.temp-ok,
    .light.breath-ok {
      fill: #16a34a;
    }
    .light.hr-warn,
    .light.rhythm-warn,
    .light.temp-warn,
    .light.breath-warn {
      fill: #dc2626;
    }
    .light.disabled {
      fill: #9ca3af;
    }
    .light.hidden {
      visibility: hidden;
    }

    button {
      border: 1px solid #234c74;
      background: #234c74;
      color: #fff;
      font-size: 12px;
      font-weight: 700;
      border-radius: 999px;
      padding: 7px 11px;
      cursor: pointer;
    }
    button.alt {
      background: #ecf3fb;
      color: #21486f;
      border-color: #c1d3e8;
    }
    .plan-wrap {
      width: 1110px;
      height: 805px;
      min-width: 1110px;
      position: relative;
      border: 1px solid #c7d8eb;
      border-radius: 12px;
      overflow: hidden;
      background:
        linear-gradient(to right, rgba(38, 72, 108, .06) 1px, transparent 1px),
        linear-gradient(to bottom, rgba(38, 72, 108, .06) 1px, transparent 1px),
        #fafdff;
      background-size: 20px 20px;
    }
    svg { width: 1110px; height: 805px; display: block; }
    .thermal-panel {
      position: absolute;
      right: 124px;
      bottom: 12px;
      width: 560px;
      padding: 10px;
      border: 1px solid rgba(155, 183, 213, .82);
      border-radius: 8px;
      background: rgba(10, 28, 46, .88);
      box-shadow: 0 9px 20px rgba(15, 34, 56, .4);
      color: #d9ecff;
      pointer-events: none;
      z-index: 4;
    }
    .thermal-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
      font-size: 12px;
      margin-bottom: 6px;
    }
    #thermal-sensor {
      font-family: "IBM Plex Mono", monospace;
      font-size: 11px;
      color: #9fd2ff;
    }
    #thermal-canvas {
      width: 100%;
      height: 390px;
      display: block;
      border: 1px solid rgba(130, 165, 199, .5);
      border-radius: 6px;
      background: #0a1d30;
    }
    .thermal-range {
      margin-top: 6px;
      min-height: 16px;
      text-align: center;
      font-size: 11px;
      color: #9fc0df;
      font-family: "IBM Plex Mono", monospace;
    }
    .thermal-timeline {
      display: grid;
      gap: 5px;
      margin-top: 7px;
      pointer-events: auto;
    }
    #thermal-history-slider {
      width: 100%;
      accent-color: #38bdf8;
      cursor: pointer;
      pointer-events: auto;
    }
    #thermal-history-slider:disabled {
      cursor: not-allowed;
      opacity: .46;
    }
    .thermal-time-row {
      display: flex;
      justify-content: space-between;
      gap: 8px;
      color: #9fc0df;
      font-family: "IBM Plex Mono", monospace;
      font-size: 10px;
      line-height: 1.2;
    }
    #thermal-time-label {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    #thermal-frame-count {
      flex: 0 0 auto;
    }
    .outline { fill: none; stroke: #1f4b71; stroke-width: 3; }
    .corridor { fill: none; stroke: none; }
    .room-shape {
      fill: var(--room);
      stroke: var(--line);
      stroke-width: 2;
      cursor: pointer;
      transition: fill .15s ease;
    }
    .room-shape:hover, .room-shape.active { fill: #def0ff; }
    .room-label {
      font-size: 24px;
      font-weight: 800;
      fill: #12365d;
      pointer-events: none;
    }
    .nurse-shape {
      fill: #eef6ff;
      stroke: #2f5d86;
      stroke-width: 2;
      stroke-dasharray: 8 5;
    }
    .nurse-label {
      font-weight: 800;
      fill: #1d4f7c;
      letter-spacing: 0;
      pointer-events: none;
    }
    .door { stroke: #2f8df5; stroke-width: 6; stroke-linecap: round; }
    .bed { cursor: pointer; }
    .bed rect {
      fill: var(--bed);
      stroke: #62819e;
      stroke-width: 1.4;
      rx: 5; ry: 5;
      transition: all .15s ease;
    }
    .bed text {
      font-size: 9px;
      fill: #18324c;
      font-weight: 700;
      pointer-events: none;
    }
    .bed.updated rect {
      animation: bed-update-pulse .42s ease-out;
    }
    @keyframes bed-update-pulse {
      0% {
        filter: drop-shadow(0 0 0 rgba(20, 184, 166, 0));
        stroke: #14b8a6;
        stroke-width: 3.2;
      }
      45% {
        filter: drop-shadow(0 0 7px rgba(20, 184, 166, .72));
        stroke: #14b8a6;
        stroke-width: 3.2;
      }
      100% {
        filter: drop-shadow(0 0 0 rgba(20, 184, 166, 0));
      }
    }
    .bed.alert rect {
      stroke: #be123c;
      stroke-width: 3.5;
    }
    .bed.out-of-bed rect {
      fill: var(--out-of-bed);
      stroke: #b7791f;
      stroke-width: 2;
    }
    .bed.alert.out-of-bed rect {
      stroke: #b7791f;
      stroke-width: 2;
    }
    .bed.offline rect {
      fill: #d1d5db;
    }
    .bed.selected rect {
      stroke: #2563eb;
      stroke-width: 3;
      filter: drop-shadow(0 0 4px rgba(37, 99, 235, 0.4));
    }
    .right {
      padding: 14px;
      display: flex;
      flex-direction: column;
      gap: 11px;
      min-height: 100%;
      background: linear-gradient(165deg, #102843, #0f2036);
      color: #e2f1ff;
      border: 1px solid #203e62;
    }
    .right h2 { margin: 0; font-size: 16px; }
    .selection-head {
      display: grid;
      grid-template-columns: minmax(118px, 1fr) auto;
      align-items: start;
      gap: 10px;
    }
    .clock { font-size: 12px; color: #aad5ff; font-family: "IBM Plex Mono", monospace; }
    #sel-meta {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
      min-width: 0;
    }
    .meta-col {
      display: grid;
      gap: 6px;
      min-width: 0;
      align-content: start;
    }
    .kv {
      display: grid;
      grid-template-columns: minmax(52px, .62fr) minmax(0, 1fr);
      gap: 6px;
      font-size: 13px;
      align-items: start;
      min-width: 0;
    }
    .kv b { color: #8cc5ff; overflow-wrap: anywhere; }
    .kv span { min-width: 0; overflow-wrap: anywhere; }
    .summary {
      border: 1px solid rgba(153, 194, 232, .35);
      border-radius: 10px;
      padding: 6px 7px;
      background: rgba(255,255,255,.06);
      font-size: 11px;
      line-height: 1.35;
      justify-self: end;
      max-width: 300px;
      white-space: nowrap;
    }
    .range-panel {
      border: 1px solid rgba(153, 194, 232, .35);
      border-radius: 10px;
      padding: 9px;
      background: rgba(255,255,255,.06);
      display: grid;
      gap: 7px;
      --el-bg-color-overlay: #0e2742;
      --el-fill-color-blank: #0e2742;
      --el-fill-color-light: #163552;
      --el-border-color-light: #486e95;
      --el-border-color-lighter: #3f6286;
      --el-text-color-primary: #e6f5ff;
      --el-text-color-regular: #c8d9ea;
      --el-text-color-placeholder: #92aeca;
      --el-color-primary: #7fb4e4;
      --el-mask-color-extra-light: rgba(255, 255, 255, .08);
    }
    .range-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 6px;
    }
    .range-field {
      display: grid;
      gap: 3px;
    }
    .range-field span {
      font-size: 11px;
      color: #9fd2ff;
    }
    .range-field .el-date-editor {
      width: 100%;
    }
    .range-field .el-input__wrapper {
      background: #0e2742;
      box-shadow: 0 0 0 1px #486e95 inset;
      border-radius: 7px;
    }
    .range-field .el-input__wrapper.is-focus {
      box-shadow: 0 0 0 1px #6aa2d6 inset;
    }
    .range-field .el-input__inner {
      color: #e6f5ff;
      font-size: 12px;
      height: 28px;
      line-height: 28px;
    }
    .range-field .el-input__prefix-inner,
    .range-field .el-input__suffix-inner {
      color: #ffffff;
    }
    .range-panel .el-picker-panel {
      background: #0e2742;
      color: #e6f5ff;
      border: 1px solid #486e95;
    }
    .range-panel .el-date-picker__header-label,
    .range-panel .el-picker-panel__icon-btn {
      color: #dcecff;
    }
    .range-panel .el-date-table th {
      color: #a8c4df;
      border-bottom-color: #35516f;
    }
    .range-panel .el-date-table td .el-date-table-cell__text {
      color: #dcecff;
    }
    .range-panel .el-date-table td.available:hover .el-date-table-cell__text {
      background: #285681;
      color: #ffffff;
    }
    .range-panel .el-date-table td.today .el-date-table-cell__text {
      color: #8bc5ff;
    }
    .range-panel .el-time-panel {
      background: #0e2742;
      border: 1px solid #486e95;
    }
    .range-panel .el-time-spinner__item {
      color: #b9cde1;
    }
    .range-panel .el-time-spinner__item.is-active {
      color: #ffffff;
      font-weight: 700;
    }
    .range-actions {
      display: flex;
      gap: 6px;
      flex-wrap: wrap;
    }
    .range-actions button {
      font-size: 11px;
      padding: 5px 9px;
    }
    .charts {
      margin-top: auto;
      display: grid;
      gap: 10px;
      grid-template-rows: repeat(3, minmax(0, 1fr));
      min-height: 500px;
      flex: 1 1 auto;
    }
    .chart-card {
      border: 1px solid rgba(153, 194, 232, .35);
      border-radius: 10px;
      padding: 9px;
      background: rgba(255,255,255,.06);
      display: grid;
      gap: 6px;
    }
    .chart-head {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      font-size: 12px;
      color: #dcefff;
    }
    .chart-now {
      color: #97d3ff;
      font-family: "IBM Plex Mono", monospace;
    }
    .chart-svg {
      width: 100%;
      height: 126px;
      display: block;
      border-radius: 7px;
      background: rgba(14, 38, 63, .38);
    }
    @media (max-width: 1220px) {
      .layout { grid-template-columns: 1fr; }
      .plan-wrap { min-width: 980px; }
      .thermal-panel {
        right: 124px;
        width: 500px;
      }
      #thermal-canvas {
        height: 350px;
      }
      .charts {
        min-height: 420px;
      }
      .chart-svg {
        height: 112px;
      }
    }
</style>
