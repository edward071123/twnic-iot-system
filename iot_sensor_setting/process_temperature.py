import argparse
import json
import os
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Optional

try:
    import psycopg2
except ModuleNotFoundError:
    psycopg2 = None

try:
    from sqlalchemy import create_engine, text
    from sqlalchemy.engine import URL
except ModuleNotFoundError:
    create_engine = None
    text = None
    URL = None

BASE_DIR = Path(__file__).resolve().parent
MPL_CACHE_DIR = BASE_DIR / ".matplotlib_cache"
MPL_CACHE_DIR.mkdir(exist_ok=True)
os.environ.setdefault("MPLCONFIGDIR", str(MPL_CACHE_DIR))

import numpy as np
import pandas as pd
from PyQt5.QtCore import QDate, QDateTime, Qt, QTime, QTimer
from PyQt5.QtWidgets import (
    QApplication,
    QCalendarWidget,
    QCheckBox,
    QComboBox,
    QDialog,
    QDialogButtonBox,
    QDoubleSpinBox,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QListWidget,
    QListWidgetItem,
    QMessageBox,
    QProgressDialog,
    QPushButton,
    QSlider,
    QTextEdit,
    QVBoxLayout,
    QWidget,
)
from matplotlib import font_manager, rcParams
from matplotlib.backends.backend_qt5agg import FigureCanvasQTAgg as FigureCanvas
from matplotlib.figure import Figure

from turning_care_algorithm import (
    TurningCareConfig,
    TurningCareRecord,
    analyze_turning_care_events,
)


DEFAULT_PG_HOST = "127.0.0.1"
DEFAULT_PG_PORT = 5435
DEFAULT_PG_USER = "postgres"
DEFAULT_PG_PASSWORD = "postgres"
DEFAULT_PG_DATABASE = "sg_tch_v3"
DEFAULT_TABLE = "sensor_datas"
DEFAULT_SENSOR_ID = "6"
DATE_TIME_DISPLAY_FORMAT = "yyyy-MM-dd HH:mm:ss"
BED_EDGE_COLUMN = "bed_edge_json"
CHINESE_FONT_CANDIDATES = (
    "Heiti TC",
    "Hiragino Sans",
    "Arial Unicode MS",
    "Songti SC",
    "PingFang TC",
    "PingFang SC",
    "Microsoft JhengHei",
    "Microsoft YaHei",
    "MingLiU",
    "PMingLiU",
    "Noto Sans CJK TC",
    "SimHei",
)
KNOWN_FRAME_SHAPES = {
    64: (8, 8),
    768: (24, 32),
}

TIME_COLUMN_CANDIDATES = (
    "timestamp",
    "created_at",
    "create_time",
    "created_time",
    "updated_at",
    "time",
    "datetime",
    "log_time",
)

APP_STYLE = """
QWidget {
    background-color: #050914;
    color: #d7e7ff;
    font-family: "Heiti TC", "Hiragino Sans", "Arial Unicode MS", "Segoe UI", sans-serif;
    font-size: 13px;
}
QLabel {
    color: #8fb7ff;
    font-weight: 600;
}
QLineEdit, QTextEdit, QDoubleSpinBox, QListWidget, QComboBox {
    background-color: #0b1220;
    color: #e6f1ff;
    border: 1px solid #21436f;
    border-radius: 6px;
    padding: 7px 9px;
    selection-background-color: #00d1ff;
    selection-color: #02111f;
}
QLineEdit:focus, QTextEdit:focus, QDoubleSpinBox:focus, QListWidget:focus, QComboBox:focus {
    border: 1px solid #00d1ff;
}
QComboBox::drop-down {
    border: 0;
    width: 28px;
}
QComboBox QAbstractItemView {
    background-color: #0b1220;
    color: #e6f1ff;
    selection-background-color: #12385c;
}
QListWidget::item {
    border-bottom: 1px solid #14243a;
    padding: 7px 4px;
}
QListWidget::item:selected {
    background-color: #12385c;
    color: #ffffff;
}
QPushButton {
    background-color: #0d2742;
    color: #eaf6ff;
    border: 1px solid #1c8fc0;
    border-radius: 6px;
    padding: 8px 14px;
    font-weight: 700;
}
QPushButton:hover {
    background-color: #12385c;
    border-color: #00d1ff;
}
QPushButton:pressed {
    background-color: #071a2c;
}
QSlider::groove:horizontal {
    height: 6px;
    background: #132238;
    border-radius: 3px;
}
QSlider::handle:horizontal {
    width: 18px;
    margin: -6px 0;
    border-radius: 9px;
    background: #00d1ff;
    border: 1px solid #7df1ff;
}
QMessageBox {
    background-color: #050914;
}
"""

FIGURE_BG = "#050914"
AXIS_BG = "#07111f"
GRID_COLOR = "#1d3557"
TEXT_COLOR = "#d7e7ff"
MUTED_TEXT_COLOR = "#8fb7ff"
CYAN = "#00d1ff"
MAGENTA = "#ff4fd8"
GREEN = "#5cffb1"
ORANGE = "#ffb84d"
RED = "#ff5f6d"
BLUE = "#6aa8ff"
VIOLET = "#b57cff"


def configure_matplotlib_fonts():
    available_fonts = {font.name for font in font_manager.fontManager.ttflist}
    fonts = [
        font_name
        for font_name in CHINESE_FONT_CANDIDATES
        if font_name in available_fonts
    ]
    fonts.extend(["DejaVu Sans", "Arial", "sans-serif"])
    rcParams["font.family"] = "sans-serif"
    rcParams["font.sans-serif"] = fonts
    rcParams["axes.unicode_minus"] = False


@dataclass
class PostgresConfig:
    host: str
    port: int
    username: str
    password: str
    database: str


@dataclass(frozen=True)
class SensorOption:
    sensor_id: int
    sensor_number: str


@dataclass
class BedEdgeContactRecord:
    start_index: int
    end_index: int
    start_time: pd.Timestamp
    end_time: pd.Timestamp
    frame_period_seconds: float

    @property
    def duration_seconds(self) -> float:
        duration = (self.end_time - self.start_time).total_seconds()
        return max(duration, self.frame_period_seconds)


def quote_identifier(name: str) -> str:
    return '"' + name.replace('"', '""') + '"'


def quote_table_name(table_name: str) -> str:
    return ".".join(quote_identifier(part.strip()) for part in table_name.split("."))


def split_table_name(table_name: str) -> tuple[Optional[str], str]:
    parts = [part.strip() for part in table_name.split(".") if part.strip()]
    if len(parts) == 2:
        return parts[0], parts[1]
    return None, table_name.strip()


def table_exists(conn, table_name: str) -> bool:
    schema_name, bare_table_name = split_table_name(table_name)
    with conn.cursor() as cursor:
        if schema_name:
            cursor.execute(
                """
                SELECT 1
                FROM information_schema.tables
                WHERE table_schema = %s
                  AND table_name = %s
                  AND table_type IN ('BASE TABLE', 'VIEW')
                """,
                (schema_name, bare_table_name),
            )
        else:
            cursor.execute(
                """
                SELECT 1
                FROM information_schema.tables
                WHERE table_schema = ANY (current_schemas(false))
                  AND table_name = %s
                  AND table_type IN ('BASE TABLE', 'VIEW')
                """,
                (bare_table_name,),
            )
        row = cursor.fetchone()
    return row is not None


def get_columns(conn, table_name: str) -> list[str]:
    schema_name, bare_table_name = split_table_name(table_name)
    with conn.cursor() as cursor:
        if schema_name:
            cursor.execute(
                """
                SELECT column_name
                FROM information_schema.columns
                WHERE table_schema = %s
                  AND table_name = %s
                ORDER BY ordinal_position
                """,
                (schema_name, bare_table_name),
            )
        else:
            cursor.execute(
                """
                SELECT column_name
                FROM information_schema.columns
                WHERE table_schema = ANY (current_schemas(false))
                  AND table_name = %s
                ORDER BY ordinal_position
                """,
                (bare_table_name,),
            )
        rows = cursor.fetchall()
    return [row[0] for row in rows]


def detect_time_column(columns: Iterable[str]) -> str:
    column_set = set(columns)
    for candidate in TIME_COLUMN_CANDIDATES:
        if candidate in column_set:
            return candidate
    raise ValueError(
        "找不到時間欄位，請確認 PostgreSQL 表格包含 timestamp/created_at/time 其中之一。"
    )


def ensure_psycopg2_available():
    if psycopg2 is None:
        raise RuntimeError(
            "缺少 PostgreSQL 套件 psycopg2。請先安裝 psycopg2-binary 或 psycopg2。"
        )


def ensure_sqlalchemy_available():
    if create_engine is None or text is None or URL is None:
        raise RuntimeError(
            "缺少 PostgreSQL 查詢套件 SQLAlchemy。請先安裝 SQLAlchemy。"
        )


def connect_postgres(config: PostgresConfig):
    ensure_psycopg2_available()
    return psycopg2.connect(
        host=config.host,
        port=config.port,
        user=config.username,
        password=config.password,
        dbname=config.database,
    )


def create_postgres_engine(config: PostgresConfig):
    ensure_sqlalchemy_available()
    url = URL.create(
        drivername="postgresql+psycopg2",
        username=config.username,
        password=config.password,
        host=config.host,
        port=config.port,
        database=config.database,
    )
    return create_engine(url)


def load_sensor_options(config: PostgresConfig) -> list[SensorOption]:
    conn = connect_postgres(config)
    try:
        if not table_exists(conn, "sensors"):
            raise ValueError("PostgreSQL 內找不到表格: sensors")

        columns = get_columns(conn, "sensors")
        required = {"sensor_id", "sensor_number"}
        missing = sorted(required - set(columns))
        if missing:
            raise ValueError(f"sensors 缺少必要欄位: {', '.join(missing)}")

        with conn.cursor() as cursor:
            cursor.execute(
                """
                SELECT sensor_id, sensor_number
                FROM sensors
                WHERE sensor_number IS NOT NULL
                ORDER BY sensor_number
                """
            )
            rows = cursor.fetchall()
    finally:
        conn.close()

    return [
        SensorOption(sensor_id=int(row[0]), sensor_number=str(row[1]))
        for row in rows
    ]


def ensure_sensor_bed_edge_column(config: PostgresConfig):
    conn = connect_postgres(config)
    try:
        with conn.cursor() as cursor:
            cursor.execute(
                f"""
                ALTER TABLE sensors
                ADD COLUMN IF NOT EXISTS {quote_identifier(BED_EDGE_COLUMN)} JSONB
                """
            )
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def load_sensor_bed_edge_points(
    config: PostgresConfig,
    sensor_id: int,
    frame_shape: tuple[int, int],
) -> list[tuple[float, float]]:
    ensure_sensor_bed_edge_column(config)
    conn = connect_postgres(config)
    try:
        with conn.cursor() as cursor:
            cursor.execute(
                f"""
                SELECT {quote_identifier(BED_EDGE_COLUMN)}
                FROM sensors
                WHERE sensor_id = %s
                """,
                (sensor_id,),
            )
            row = cursor.fetchone()
    finally:
        conn.close()

    if not row or row[0] in (None, ""):
        return []

    payload = row[0]
    if isinstance(payload, str):
        payload = json.loads(payload)
    _closed, points = normalize_bed_edge_payload(payload, frame_shape)
    return points


def save_sensor_bed_edge_points(
    config: PostgresConfig,
    sensor_id: int,
    payload: dict,
):
    ensure_sensor_bed_edge_column(config)
    conn = connect_postgres(config)
    try:
        with conn.cursor() as cursor:
            cursor.execute(
                f"""
                UPDATE sensors
                SET {quote_identifier(BED_EDGE_COLUMN)} = %s::jsonb
                WHERE sensor_id = %s
                """,
                (json.dumps(payload, ensure_ascii=False), sensor_id),
            )
            if cursor.rowcount != 1:
                raise ValueError(f"sensors 找不到 sensor_id={sensor_id}")
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def parse_temperature_json(value) -> Optional[np.ndarray]:
    if value is None or value == "":
        return None

    if isinstance(value, str):
        value = json.loads(value)

    if isinstance(value, dict):
        for key in ("temperature", "temperatures", "data", "values", "pixels"):
            if key in value:
                value = value[key]
                break

    array = np.asarray(value, dtype=float).flatten()
    frame_shape = KNOWN_FRAME_SHAPES.get(array.size)
    if frame_shape is None:
        side = int(np.sqrt(array.size))
        if side * side == array.size:
            frame_shape = (side, side)
        else:
            expected_sizes = ", ".join(str(size) for size in KNOWN_FRAME_SHAPES)
            raise ValueError(
                f"熱像 frame 長度為 {array.size}，不是已知熱像尺寸 ({expected_sizes})"
            )

    return array.reshape(frame_shape)


def detect_thermal_json_column(columns: Iterable[str]) -> str:
    for column_name in ("temperature_json", "frame_json"):
        if column_name in columns:
            return column_name
    raise ValueError("資料表缺少熱像欄位: frame_json 或 temperature_json")


def detect_high_temperature_column(columns: Iterable[str]) -> Optional[str]:
    column_set = set(columns)
    if "high_temperature" in column_set:
        return "high_temperature"
    return None


class DateTimePicker(QWidget):
    def __init__(self, value: QDateTime, parent: Optional[QWidget] = None):
        super().__init__(parent)
        self._value = value

        self.display = QLineEdit(self)
        self.display.setReadOnly(True)
        self.button = QPushButton("選擇", self)
        self.button.clicked.connect(self.open_picker)

        layout = QHBoxLayout()
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(8)
        layout.addWidget(self.display, 1)
        layout.addWidget(self.button)
        self.setLayout(layout)
        self.update_display()

    def dateTime(self) -> QDateTime:
        return self._value

    def setDateTime(self, value: QDateTime):
        self._value = value
        self.update_display()

    def update_display(self):
        self.display.setText(self._value.toString(DATE_TIME_DISPLAY_FORMAT))

    def create_number_combo(self, start: int, end: int, value: int) -> QComboBox:
        combo = QComboBox(self)
        for number in range(start, end + 1):
            combo.addItem(f"{number:02d}", number)
        index = combo.findData(value)
        if index >= 0:
            combo.setCurrentIndex(index)
        return combo

    def open_picker(self):
        dialog = QDialog(self)
        dialog.setWindowTitle("選擇日期時間")
        dialog.setStyleSheet(APP_STYLE)

        calendar = QCalendarWidget(dialog)
        calendar.setSelectedDate(self._value.date())
        calendar.setGridVisible(True)

        current_time = self._value.time()
        hour_combo = self.create_number_combo(0, 23, current_time.hour())
        minute_combo = self.create_number_combo(0, 59, current_time.minute())
        second_combo = self.create_number_combo(0, 59, current_time.second())

        time_layout = QHBoxLayout()
        time_layout.addWidget(QLabel("時"))
        time_layout.addWidget(hour_combo)
        time_layout.addWidget(QLabel("分"))
        time_layout.addWidget(minute_combo)
        time_layout.addWidget(QLabel("秒"))
        time_layout.addWidget(second_combo)

        buttons = QDialogButtonBox(
            QDialogButtonBox.Ok | QDialogButtonBox.Cancel,
            Qt.Horizontal,
            dialog,
        )
        buttons.accepted.connect(dialog.accept)
        buttons.rejected.connect(dialog.reject)

        layout = QVBoxLayout()
        layout.addWidget(calendar)
        layout.addWidget(QLabel("時間"))
        layout.addLayout(time_layout)
        layout.addWidget(buttons)
        dialog.setLayout(layout)

        if dialog.exec_() == QDialog.Accepted:
            selected_time = QTime(
                int(hour_combo.currentData()),
                int(minute_combo.currentData()),
                int(second_combo.currentData()),
            )
            self.setDateTime(QDateTime(calendar.selectedDate(), selected_time))


def today_datetime_range() -> tuple[QDateTime, QDateTime]:
    today = QDate.currentDate()
    return (
        QDateTime(today, QTime(0, 0, 0)),
        QDateTime(today, QTime(23, 59, 59)),
    )


def normalize_bed_edge_payload(payload, frame_shape: tuple[int, int]) -> tuple[bool, list[tuple[float, float]]]:
    if not isinstance(payload, dict):
        raise ValueError("床緣 JSON 必須是物件格式。")

    raw_points = payload.get("points")
    if not isinstance(raw_points, list):
        raise ValueError("床緣 JSON 缺少 points 清單。")

    height, width = frame_shape
    points: list[tuple[float, float]] = []
    for raw_point in raw_points:
        if not isinstance(raw_point, (list, tuple)) or len(raw_point) != 2:
            raise ValueError("每個床緣點必須是 [x, y]。")
        x = min(max(float(raw_point[0]), 0.0), width - 1)
        y = min(max(float(raw_point[1]), 0.0), height - 1)
        points.append((round(x, 2), round(y, 2)))

    closed = bool(payload.get("closed", False)) and len(points) >= 3
    return closed, points


def paired_bed_edge_segments(
    points: list[tuple[float, float]]
) -> list[tuple[tuple[float, float], tuple[float, float]]]:
    return [
        (points[index], points[index + 1])
        for index in range(0, len(points) - 1, 2)
    ]


def min_distance_to_segments(
    points: np.ndarray,
    segments: list[tuple[tuple[float, float], tuple[float, float]]],
) -> np.ndarray:
    if points.size == 0 or not segments:
        return np.array([], dtype=float)

    distances = np.full(points.shape[0], np.inf, dtype=float)
    for start, end in segments:
        segment_start = np.asarray(start, dtype=float)
        segment_end = np.asarray(end, dtype=float)
        segment = segment_end - segment_start
        segment_length_sq = float(np.dot(segment, segment))
        if segment_length_sq <= 1e-9:
            candidate = np.linalg.norm(points - segment_start, axis=1)
        else:
            t = np.clip(((points - segment_start) @ segment) / segment_length_sq, 0.0, 1.0)
            projection = segment_start + t[:, None] * segment
            candidate = np.linalg.norm(points - projection, axis=1)
        distances = np.minimum(distances, candidate)
    return distances


def style_axis(ax, title: str, xlabel: Optional[str] = None, ylabel: Optional[str] = None):
    ax.set_facecolor(AXIS_BG)
    ax.set_title(title, color=TEXT_COLOR, pad=8, fontweight="bold")
    if xlabel:
        ax.set_xlabel(xlabel, color=MUTED_TEXT_COLOR)
    if ylabel:
        ax.set_ylabel(ylabel, color=MUTED_TEXT_COLOR)
    ax.tick_params(colors=MUTED_TEXT_COLOR)
    for spine in ax.spines.values():
        spine.set_color("#21436f")


def load_frames_from_postgres(
    config: PostgresConfig,
    table_name: str,
    sensor_id: int,
    start: pd.Timestamp,
    end: pd.Timestamp,
) -> tuple[list[np.ndarray], list[pd.Timestamp], int]:
    conn = connect_postgres(config)
    try:
        if not table_exists(conn, table_name):
            raise ValueError(f"PostgreSQL 內找不到表格: {table_name}")

        columns = get_columns(conn, table_name)
        required = {"sensor_id"}
        missing = sorted(required - set(columns))
        if missing:
            raise ValueError(f"{table_name} 缺少必要欄位: {', '.join(missing)}")

        thermal_column = detect_thermal_json_column(columns)
        high_temperature_column = detect_high_temperature_column(columns)
        time_column = detect_time_column(columns)
        high_temperature_select = (
            f"{quote_identifier(high_temperature_column)} AS high_temperature"
            if high_temperature_column
            else "NULL AS high_temperature"
        )
        sql = f"""
            SELECT {quote_identifier(thermal_column)} AS thermal_json,
                   {high_temperature_select},
                   {quote_identifier(time_column)} AS frame_time
            FROM {quote_table_name(table_name)}
            WHERE sensor_id = :sensor_id
              AND {quote_identifier(time_column)} >= :start_time
              AND {quote_identifier(time_column)} < :end_time
            ORDER BY {quote_identifier(time_column)}
        """
    finally:
        conn.close()

    engine = create_postgres_engine(config)
    try:
        with engine.connect() as sqlalchemy_conn:
            df = pd.read_sql_query(
                text(sql),
                sqlalchemy_conn,
                params={
                    "sensor_id": sensor_id,
                    "start_time": start.to_pydatetime(),
                    "end_time": end.to_pydatetime(),
                },
            )
    finally:
        engine.dispose()

    frames: list[np.ndarray] = []
    times: list[pd.Timestamp] = []
    skipped = 0
    expected_shape = None

    for row in df.itertuples(index=False):
        try:
            high_temperature = getattr(row, "high_temperature", None)
            frame = parse_temperature_json(row.thermal_json)
            if frame is None:
                skipped += 1
                continue
            if high_temperature is None:
                print("警告: 資料列缺少 high_temperature，仍使用 temperature_json 繪製熱像。")
            if expected_shape is None:
                expected_shape = frame.shape
            elif frame.shape != expected_shape:
                skipped += 1
                print(f"略過不同熱像尺寸: {frame.shape}，預期 {expected_shape}")
                continue
            frames.append(frame)
            times.append(pd.to_datetime(row.frame_time))
        except Exception as exc:
            skipped += 1
            print(f"解析熱像 frame 失敗: {exc}")

    return frames, times, skipped


class HeatmapWindow(QWidget):
    MIN_CARE_MOTION_SECONDS = 5.0
    CARE_EVENT_GAP_SECONDS = 0.75
    CARE_RECORD_MERGE_GAP_SECONDS = 8.0
    CARE_RECORD_PADDING_SECONDS = 3.0
    BED_EDGE_CONTACT_MERGE_GAP_SECONDS = 600.0
    MIN_BED_EDGE_CONTACT_SECONDS = 10.0
    DEFAULT_MOTION_THRESHOLD_SCORE = 3.0
    DEFAULT_SCORE_SMOOTH_SECONDS = 60.0
    WAVE_BASE_RATIO = 0.18
    TURN_CENTROID_DISPLACEMENT_MIN = 1.8
    TURN_CHANGED_AREA_MIN = 0.075
    TURN_CHANGED_WIDTH_MIN = 22.0
    TURN_CHANGED_HEIGHT_MIN = 18.0
    TURN_TEMP_MEAN_DELTA_MIN = 0.65
    REPOSITION_CHANGED_AREA_MIN = 0.075
    REPOSITION_CHANGED_WIDTH_MIN = 22.0
    REPOSITION_CHANGED_HEIGHT_MIN = 18.0
    REPOSITION_CENTROID_PATH_MIN = 35.0
    REPOSITION_WARM_AREA_MIN = 0.20
    REPOSITION_COOL_AREA_MIN = 0.08
    TURNING_CARE_MERGE_GAP_SECONDS = 600.0

    def __init__(
        self,
        frames,
        times,
        title,
        pg_config: PostgresConfig,
        sensor_id: int,
        sensor_number: str,
    ):
        super().__init__()
        self.frames = frames
        self.times = times
        self.index = 0
        self.pg_config = pg_config
        self.sensor_id = sensor_id
        self.sensor_number = sensor_number

        self.setWindowTitle(title)
        self.resize(1250, 980)
        self.setStyleSheet(APP_STYLE)

        layout = QHBoxLayout()
        control_panel = QWidget()
        control_panel.setMaximumWidth(620)
        control_panel.setMinimumWidth(560)
        control_panel.setStyleSheet(APP_STYLE)
        left_layout = QVBoxLayout(control_panel)
        right_layout = QVBoxLayout()

        self.canvas = FigureCanvas(Figure(facecolor=FIGURE_BG))
        self.canvas.setMinimumHeight(650)
        self.canvas.setMinimumWidth(700)
        right_layout.addWidget(self.canvas, stretch=1)
        layout.addWidget(control_panel, stretch=1)
        layout.addLayout(right_layout, stretch=3)

        grid = self.canvas.figure.add_gridspec(2, 1, height_ratios=[3.2, 1.0])
        self.ax_heatmap = self.canvas.figure.add_subplot(grid[0])
        self.ax_chart = self.canvas.figure.add_subplot(grid[1])
        self.canvas.figure.subplots_adjust(
            hspace=0.22,
            left=0.08,
            right=0.94,
            top=0.94,
            bottom=0.12,
        )

        self.im = self.ax_heatmap.imshow(
            np.zeros(frames[0].shape),
            cmap="inferno",
            origin="lower",
            aspect="equal",
        )
        self.colorbar = self.canvas.figure.colorbar(self.im, ax=self.ax_heatmap, label="Temperature (C)")
        self.colorbar.ax.yaxis.label.set_color(MUTED_TEXT_COLOR)
        self.colorbar.ax.tick_params(colors=MUTED_TEXT_COLOR)
        self.colorbar.outline.set_edgecolor("#21436f")
        style_axis(self.ax_heatmap, "Thermal Heatmap")
        self.ax_heatmap.set_xticks([])
        self.ax_heatmap.set_yticks([])

        self.motion_scores: np.ndarray = np.array([])
        self.care_scores: np.ndarray = np.array([])
        self.care_indices: set[int] = set()
        self.care_event_ranges: list[tuple[int, int]] = []
        self.care_record_event_ranges: list[tuple[int, int]] = []
        self.care_records: list[TurningCareRecord] = []
        self.bed_edge_contact_records: list[BedEdgeContactRecord] = []
        self.event_list_items: list[tuple[str, int]] = []
        self.cursor_line = None
        self.chart_pan_start_x = None
        self.chart_pan_start_xlim = None
        self.bed_edge_points: list[tuple[float, float]] = []
        self.bed_edge_line = None
        self.bed_edge_scatter = None
        self.edge_alert_text = None
        self.edge_contact_scatter = None
        self.drawing_enabled = True
        self.frame_period_seconds = self.estimate_frame_period_seconds()
        self.motion_threshold_score = self.DEFAULT_MOTION_THRESHOLD_SCORE
        self.score_smooth_seconds = self.DEFAULT_SCORE_SMOOTH_SECONDS
        self.disable_score_smoothing = False

        self.calculate_analysis()
        self.generate_care_records()
        self.draw_analysis_chart()

        self.canvas.mpl_connect("button_press_event", self.on_canvas_button_press)
        self.canvas.mpl_connect("button_release_event", self.on_canvas_button_release)
        self.canvas.mpl_connect("motion_notify_event", self.on_canvas_motion)
        self.canvas.mpl_connect("scroll_event", self.on_canvas_scroll)

        left_layout.addWidget(QLabel("床緣標記工具"))
        edge_btn_layout = QHBoxLayout()
        undo_edge_btn = QPushButton("復原上一點")
        clear_edge_btn = QPushButton("清除床緣")
        save_edge_btn = QPushButton("儲存到 Sensor DB")
        undo_edge_btn.clicked.connect(self.undo_bed_edge_point)
        clear_edge_btn.clicked.connect(self.clear_bed_edge_points)
        save_edge_btn.clicked.connect(self.save_bed_edge_points)
        edge_btn_layout.addWidget(undo_edge_btn)
        edge_btn_layout.addWidget(clear_edge_btn)
        edge_btn_layout.addWidget(save_edge_btn)
        left_layout.addLayout(edge_btn_layout)

        edge_temp_layout = QHBoxLayout()
        edge_temp_layout.addWidget(QLabel("邊緣接觸判斷溫度 (C)"))
        self.edge_contact_temp_spin = QDoubleSpinBox()
        self.edge_contact_temp_spin.setRange(20.0, 45.0)
        self.edge_contact_temp_spin.setSingleStep(0.5)
        self.edge_contact_temp_spin.setDecimals(1)
        self.edge_contact_temp_spin.setValue(28.0)
        self.edge_contact_temp_spin.valueChanged.connect(self.bed_edge_contact_setting_changed)
        edge_temp_layout.addWidget(self.edge_contact_temp_spin)
        left_layout.addLayout(edge_temp_layout)

        motion_threshold_layout = QHBoxLayout()
        motion_threshold_layout.addWidget(QLabel("Motion 閥值"))
        self.motion_threshold_spin = QDoubleSpinBox()
        self.motion_threshold_spin.setRange(0.1, 20.0)
        self.motion_threshold_spin.setSingleStep(0.1)
        self.motion_threshold_spin.setDecimals(2)
        self.motion_threshold_spin.setValue(self.motion_threshold_score)
        motion_threshold_layout.addWidget(self.motion_threshold_spin)
        apply_motion_threshold_btn = QPushButton("設定")
        apply_motion_threshold_btn.clicked.connect(self.apply_motion_threshold)
        motion_threshold_layout.addWidget(apply_motion_threshold_btn)
        left_layout.addLayout(motion_threshold_layout)

        smooth_layout = QHBoxLayout()
        smooth_layout.addWidget(QLabel("均化秒數"))
        self.score_smooth_spin = QDoubleSpinBox()
        self.score_smooth_spin.setRange(0.5, 99999.0)
        self.score_smooth_spin.setSingleStep(0.5)
        self.score_smooth_spin.setDecimals(1)
        self.score_smooth_spin.setValue(self.score_smooth_seconds)
        smooth_layout.addWidget(self.score_smooth_spin)
        self.disable_smoothing_checkbox = QCheckBox("取消均化")
        self.disable_smoothing_checkbox.setChecked(self.disable_score_smoothing)
        smooth_layout.addWidget(self.disable_smoothing_checkbox)
        left_layout.addLayout(smooth_layout)

        self.edge_box = QTextEdit()
        self.edge_box.setReadOnly(True)
        self.edge_box.setMinimumHeight(85)
        self.edge_box.setText("點選熱像圖標記 4 個床緣座標，完成後儲存到 sensors。")
        left_layout.addWidget(self.edge_box)

        self.edge_alert_label = QLabel("")
        self.edge_alert_label.setStyleSheet(
            "color: #ffffff; background-color: #7a1020; border: 1px solid #ffb4bd; "
            "border-radius: 6px; padding: 8px; font-weight: 700;"
        )
        self.edge_alert_label.setVisible(False)
        left_layout.addWidget(self.edge_alert_label)

        left_layout.addWidget(QLabel("護理師照護時間分析"))
        self.record_summary_label = QLabel("")
        left_layout.addWidget(self.record_summary_label)
        self.record_list = QListWidget()
        self.record_list.setMinimumHeight(150)
        self.record_list.currentRowChanged.connect(self.focus_care_record)
        left_layout.addWidget(self.record_list, stretch=1)
        self.populate_care_record_list()

        self.slider = QSlider(Qt.Horizontal)
        self.slider.setMinimum(0)
        self.slider.setMaximum(len(frames) - 1)
        self.slider.valueChanged.connect(self.slider_changed)
        left_layout.addWidget(self.slider)

        btn_layout = QHBoxLayout()
        prev_btn = QPushButton("上一張")
        play_btn = QPushButton("播放")
        stop_btn = QPushButton("停止")
        next_btn = QPushButton("下一張")
        prev_btn.clicked.connect(self.prev_manual)
        play_btn.clicked.connect(self.play)
        stop_btn.clicked.connect(self.stop)
        next_btn.clicked.connect(self.next_manual)
        btn_layout.addWidget(prev_btn)
        btn_layout.addWidget(play_btn)
        btn_layout.addWidget(stop_btn)
        btn_layout.addWidget(next_btn)
        left_layout.addLayout(btn_layout)
        left_layout.addStretch(1)

        self.setLayout(layout)

        self.timer = QTimer()
        self.timer.timeout.connect(self.next_frame)
        self.load_bed_edge_points_from_db(show_message=False)
        self.update_frame(0)

    def estimate_frame_period_seconds(self) -> float:
        if len(self.times) < 2:
            return 0.0

        diffs = pd.Series(self.times).diff().dt.total_seconds().dropna()
        diffs = diffs[diffs > 0]
        if diffs.empty:
            return 0.0
        return float(diffs.median())

    def on_canvas_button_press(self, event):
        if event.inaxes == self.ax_heatmap:
            self.on_heatmap_click(event)
            return

        if event.inaxes == self.ax_chart:
            if event.button == 1 and event.xdata is not None:
                idx = int(round(event.xdata))
                idx = min(max(idx, 0), len(self.frames) - 1)
                self.slider.setValue(idx)
            elif event.button in (2, 3) and event.xdata is not None:
                self.chart_pan_start_x = float(event.xdata)
                self.chart_pan_start_xlim = self.ax_chart.get_xlim()

    def on_canvas_button_release(self, _event):
        self.chart_pan_start_x = None
        self.chart_pan_start_xlim = None

    def on_canvas_motion(self, event):
        if (
            event.inaxes != self.ax_chart
            or self.chart_pan_start_x is None
            or self.chart_pan_start_xlim is None
            or event.xdata is None
        ):
            return

        dx = self.chart_pan_start_x - float(event.xdata)
        left, right = self.chart_pan_start_xlim
        self.set_chart_xlim(left + dx, right + dx)

    def on_canvas_scroll(self, event):
        if event.inaxes != self.ax_chart or event.xdata is None:
            return

        left, right = self.ax_chart.get_xlim()
        width = right - left
        if width <= 1:
            return

        scale = 0.8 if event.button == "up" else 1.25
        new_width = max(10.0, min(len(self.frames), width * scale))
        center = float(event.xdata)
        ratio = (center - left) / width
        new_left = center - new_width * ratio
        new_right = new_left + new_width
        self.set_chart_xlim(new_left, new_right)

    def set_chart_xlim(self, left: float, right: float):
        max_right = max(1, len(self.frames) - 1)
        width = max(10.0, right - left)
        if width >= max_right:
            left, right = 0, max_right
        else:
            left = max(0.0, min(left, max_right - width))
            right = left + width

        self.ax_chart.set_xlim(left, right)
        self.canvas.draw_idle()

    def on_heatmap_click(self, event):
        if not self.drawing_enabled or event.inaxes != self.ax_heatmap:
            return
        if event.xdata is None or event.ydata is None:
            return
        if len(self.bed_edge_points) >= 4:
            QMessageBox.information(
                self,
                "病床邊緣",
                "已標記 4 個床緣座標。若要重設，請先復原或清除床緣。",
            )
            return

        height, width = self.frames[0].shape
        x = min(max(float(event.xdata), 0.0), width - 1)
        y = min(max(float(event.ydata), 0.0), height - 1)
        self.bed_edge_points.append((round(x, 2), round(y, 2)))
        self.refresh_bed_edge_overlay()
        self.refresh_bed_edge_contact_events()

    def undo_bed_edge_point(self):
        if self.bed_edge_points:
            self.bed_edge_points.pop()
        self.refresh_bed_edge_overlay()
        self.refresh_bed_edge_contact_events()

    def clear_bed_edge_points(self):
        self.bed_edge_points = []
        self.refresh_bed_edge_overlay()
        self.refresh_bed_edge_contact_events()

    def bed_edge_payload(self) -> dict:
        return {
            "closed": False,
            "mode": "paired_segments",
            "frame_shape": list(self.frames[0].shape),
            "points": self.bed_edge_points,
        }

    def save_bed_edge_points(self):
        if not self.bed_edge_points:
            QMessageBox.information(self, "病床邊緣", "尚未標記床緣，沒有可儲存的座標。")
            return
        if len(self.bed_edge_points) != 4:
            QMessageBox.warning(self, "病床邊緣", "請標記 4 個床緣座標後再儲存。")
            return

        try:
            save_sensor_bed_edge_points(
                self.pg_config,
                self.sensor_id,
                self.bed_edge_payload(),
            )
        except Exception as exc:
            QMessageBox.critical(self, "資料庫錯誤", f"儲存床緣座標失敗:\n{exc}")
            return

        self.edge_box.setText(self.format_bed_edge_points())
        QMessageBox.information(
            self,
            "病床邊緣",
            f"已儲存 4 個床緣座標到 sensors.{BED_EDGE_COLUMN}\nSensor: {self.sensor_number}",
        )

    def load_bed_edge_points_from_db(self, show_message: bool):
        try:
            points = load_sensor_bed_edge_points(
                self.pg_config,
                self.sensor_id,
                self.frames[0].shape,
            )
        except Exception as exc:
            if show_message:
                QMessageBox.critical(self, "資料庫錯誤", f"載入床緣座標失敗:\n{exc}")
            return

        if not points:
            self.bed_edge_points = []
            if show_message:
                QMessageBox.information(
                    self,
                    "病床邊緣",
                    f"Sensor {self.sensor_number} 尚未儲存床緣座標。",
                )
            self.refresh_bed_edge_overlay()
            self.refresh_bed_edge_contact_events()
            return

        self.bed_edge_points = points
        self.refresh_bed_edge_overlay()
        self.refresh_bed_edge_contact_events()
        if hasattr(self, "index"):
            self.update_frame(self.index)
        if show_message:
            QMessageBox.information(
                self,
                "病床邊緣",
                f"已從 sensors.{BED_EDGE_COLUMN} 載入 {len(points)} 個床緣座標。",
            )

    def refresh_bed_edge_overlay(self):
        if self.bed_edge_line is not None:
            self.bed_edge_line.remove()
            self.bed_edge_line = None
        if self.bed_edge_scatter is not None:
            self.bed_edge_scatter.remove()
            self.bed_edge_scatter = None
        if len(self.bed_edge_points) < 2:
            self.clear_bed_edge_contact_alert()

        if self.bed_edge_points:
            xs = []
            ys = []
            for start, end in paired_bed_edge_segments(self.bed_edge_points):
                xs.extend([start[0], end[0], np.nan])
                ys.extend([start[1], end[1], np.nan])
            self.bed_edge_line = self.ax_heatmap.plot(
                xs,
                ys,
                color=CYAN,
                linewidth=2.2,
                marker="o",
                markersize=5,
                markerfacecolor=ORANGE,
                markeredgecolor="#ffffff",
            )[0]
            self.bed_edge_scatter = self.ax_heatmap.scatter(
                [point[0] for point in self.bed_edge_points],
                [point[1] for point in self.bed_edge_points],
                s=42,
                c=ORANGE,
                edgecolors="#ffffff",
                linewidths=0.8,
                zorder=5,
            )

        self.edge_box.setText(self.format_bed_edge_points())
        self.canvas.draw_idle()

    def format_bed_edge_points(self) -> str:
        if not self.bed_edge_points:
            return "尚未標記床緣。點選熱像圖即可新增座標。"

        lines = [
            f"Sensor {self.sensor_number} 床緣座標: {len(self.bed_edge_points)} 點，完整線段: {len(paired_bed_edge_segments(self.bed_edge_points))} 條",
            json.dumps(
                {
                    "closed": False,
                    "mode": "paired_segments",
                    "points": self.bed_edge_points,
                },
                ensure_ascii=False,
            ),
            "",
        ]
        if len(self.bed_edge_points) != 4:
            lines.append("儲存到 sensors.bed_edge_json 前需剛好 4 點。")
            lines.append("")
        if len(self.bed_edge_points) % 2 == 1:
            lines.append("最後 1 點尚未成線，請再點 1 點完成線段。")
            lines.append("")
        for index, (x, y) in enumerate(self.bed_edge_points, start=1):
            lines.append(f"{index}. x={x:.2f}, y={y:.2f}")
        return "\n".join(lines)

    def detect_bed_edge_contact(self, frame: np.ndarray) -> tuple[bool, np.ndarray]:
        segments = paired_bed_edge_segments(self.bed_edge_points)
        if not segments:
            return False, np.empty((0, 2), dtype=float)

        contact_temp = self.get_edge_contact_temperature()
        hot_yx = np.argwhere(frame >= contact_temp)
        if hot_yx.size == 0:
            return False, np.empty((0, 2), dtype=float)

        hot_points = np.column_stack((hot_yx[:, 1], hot_yx[:, 0])).astype(float)
        distances = min_distance_to_segments(hot_points, segments)
        near_edge = distances <= 0.85
        contact_points = hot_points[near_edge]
        min_required = max(1, min(4, int(round(len(hot_points) * 0.05))))
        return len(contact_points) >= min_required, contact_points

    def get_edge_contact_temperature(self) -> float:
        if hasattr(self, "edge_contact_temp_spin"):
            return float(self.edge_contact_temp_spin.value())
        return 28.0

    def bed_edge_contact_setting_changed(self, _value):
        self.refresh_bed_edge_contact_events()
        self.update_frame(self.index)

    def refresh_bed_edge_contact_events(self):
        self.generate_bed_edge_contact_records()
        if hasattr(self, "record_list"):
            self.populate_care_record_list()
        if hasattr(self, "ax_chart"):
            self.draw_analysis_chart()

    def generate_bed_edge_contact_records(self):
        # Uses self.bed_edge_points, which is loaded from sensors.bed_edge_json.
        # This produces in-memory edge-contact events for the lower-left list.
        self.bed_edge_contact_records = []
        if not paired_bed_edge_segments(self.bed_edge_points):
            return

        contact_indices = [
            idx
            for idx, frame in enumerate(self.frames)
            if self.detect_bed_edge_contact(frame)[0]
        ]
        if not contact_indices:
            return

        # Bed-edge contact may flicker when the temperature moves above/below the
        # threshold. Merge contacts separated by this range into one edge_alert.
        merge_gap_frames = max(
            1,
            int(round(self.BED_EDGE_CONTACT_MERGE_GAP_SECONDS / max(self.frame_period_seconds, 0.1))),
        )
        start_idx = contact_indices[0]
        prev_idx = contact_indices[0]
        for idx in contact_indices[1:]:
            if idx - prev_idx <= merge_gap_frames:
                prev_idx = idx
                continue
            self.append_bed_edge_contact_record(start_idx, prev_idx)
            start_idx = idx
            prev_idx = idx

        self.append_bed_edge_contact_record(start_idx, prev_idx)

    def append_bed_edge_contact_record(self, start_idx: int, end_idx: int):
        # Ignore short edge contacts as transient spikes.
        if self.motion_duration_seconds(start_idx, end_idx) < self.MIN_BED_EDGE_CONTACT_SECONDS:
            return

        self.bed_edge_contact_records.append(
            BedEdgeContactRecord(
                start_index=start_idx,
                end_index=end_idx,
                start_time=self.times[start_idx],
                end_time=self.times[end_idx],
                frame_period_seconds=self.frame_period_seconds,
            )
        )

    def clear_bed_edge_contact_alert(self):
        if self.edge_alert_text is not None:
            self.edge_alert_text.remove()
            self.edge_alert_text = None
        if self.edge_contact_scatter is not None:
            self.edge_contact_scatter.remove()
            self.edge_contact_scatter = None
        if hasattr(self, "edge_alert_label"):
            self.edge_alert_label.setVisible(False)
            self.edge_alert_label.setText("")

    def update_bed_edge_contact_alert(self, frame: np.ndarray):
        self.clear_bed_edge_contact_alert()
        is_contact, contact_points = self.detect_bed_edge_contact(frame)
        if not is_contact:
            return

        if hasattr(self, "edge_alert_label"):
            self.edge_alert_label.setText("警示：病患接觸到病床邊緣")
            self.edge_alert_label.setVisible(True)

        self.edge_alert_text = self.ax_heatmap.text(
            0.5,
            0.90,
            "警示：病患接觸到病床邊緣",
            transform=self.ax_heatmap.transAxes,
            ha="center",
            va="top",
            color="#ffffff",
            fontsize=14,
            fontweight="bold",
            bbox={
                "boxstyle": "round,pad=0.45",
                "facecolor": RED,
                "edgecolor": "#ffffff",
                "linewidth": 1.2,
                "alpha": 0.92,
            },
            zorder=10,
        )
        self.edge_contact_scatter = self.ax_heatmap.scatter(
            contact_points[:, 0],
            contact_points[:, 1],
            s=70,
            c=RED,
            marker="x",
            linewidths=2.0,
            zorder=9,
        )

    def calculate_analysis(self):
        analysis = analyze_turning_care_events(
            self.frames,
            self.times,
            frame_period_seconds=self.frame_period_seconds,
            motion_threshold_score=self.get_motion_threshold_score(),
            score_smooth_seconds=float(getattr(self, "score_smooth_seconds", self.DEFAULT_SCORE_SMOOTH_SECONDS)),
            disable_score_smoothing=self.disable_score_smoothing,
            config=self.turning_care_config(),
        )
        self.motion_scores = analysis.motion_scores
        self.care_scores = analysis.care_scores
        self.care_indices = analysis.care_indices
        self.care_event_ranges = analysis.care_event_ranges
        self.care_record_event_ranges = analysis.care_record_event_ranges
        self.care_records = analysis.care_records
        max_care_score = float(np.max(self.care_scores)) if self.care_scores.size else 0.0
        start_time = self.times[0] if self.times else "N/A"
        end_time = self.times[-1] if self.times else "N/A"
        print(
            "Care analysis:",
            f"frames={len(self.frames)}",
            f"range={start_time}~{end_time}",
            f"care_frame_count={len(self.care_indices)}",
            f"care_records={len(self.care_records)}",
            f"max_care_score={max_care_score:.3f}",
            f"threshold={self.get_motion_threshold_score():.3f}",
            f"smooth_seconds={float(getattr(self, 'score_smooth_seconds', self.DEFAULT_SCORE_SMOOTH_SECONDS)):.1f}",
            f"frame_period_seconds={self.frame_period_seconds:.3f}",
        )

    def turning_care_config(self) -> TurningCareConfig:
        return TurningCareConfig(
            min_care_motion_seconds=self.MIN_CARE_MOTION_SECONDS,
            care_event_gap_seconds=self.CARE_EVENT_GAP_SECONDS,
            care_record_merge_gap_seconds=self.CARE_RECORD_MERGE_GAP_SECONDS,
            care_record_padding_seconds=self.CARE_RECORD_PADDING_SECONDS,
            default_motion_threshold_score=self.DEFAULT_MOTION_THRESHOLD_SCORE,
            default_score_smooth_seconds=self.DEFAULT_SCORE_SMOOTH_SECONDS,
            wave_base_ratio=self.WAVE_BASE_RATIO,
            turn_centroid_displacement_min=self.TURN_CENTROID_DISPLACEMENT_MIN,
            turn_changed_area_min=self.TURN_CHANGED_AREA_MIN,
            turn_changed_width_min=self.TURN_CHANGED_WIDTH_MIN,
            turn_changed_height_min=self.TURN_CHANGED_HEIGHT_MIN,
            turn_temp_mean_delta_min=self.TURN_TEMP_MEAN_DELTA_MIN,
            reposition_changed_area_min=self.REPOSITION_CHANGED_AREA_MIN,
            reposition_changed_width_min=self.REPOSITION_CHANGED_WIDTH_MIN,
            reposition_changed_height_min=self.REPOSITION_CHANGED_HEIGHT_MIN,
            reposition_centroid_path_min=self.REPOSITION_CENTROID_PATH_MIN,
            reposition_warm_area_min=self.REPOSITION_WARM_AREA_MIN,
            reposition_cool_area_min=self.REPOSITION_COOL_AREA_MIN,
            turning_care_merge_gap_seconds=self.TURNING_CARE_MERGE_GAP_SECONDS,
        )

    def score_smooth_window(self) -> int:
        if self.disable_score_smoothing:
            return 1

        smooth_seconds = float(getattr(self, "score_smooth_seconds", self.DEFAULT_SCORE_SMOOTH_SECONDS))
        return max(1, int(round(smooth_seconds / max(self.frame_period_seconds, 0.1))))

    def get_motion_threshold_score(self) -> float:
        return float(getattr(self, "motion_threshold_score", self.DEFAULT_MOTION_THRESHOLD_SCORE))

    def apply_motion_threshold(self):
        self.motion_threshold_score = float(self.motion_threshold_spin.value())
        self.score_smooth_seconds = float(self.score_smooth_spin.value())
        self.disable_score_smoothing = self.disable_smoothing_checkbox.isChecked()
        current_row = self.record_list.currentRow() if hasattr(self, "record_list") else -1
        self.calculate_analysis()
        self.generate_care_records()
        self.draw_analysis_chart()
        self.populate_care_record_list()
        if 0 <= current_row < self.record_list.count():
            self.record_list.setCurrentRow(current_row)
        self.update_frame(self.index)

    def motion_duration_seconds(self, start_idx: int, end_idx: int) -> float:
        if not self.times:
            return 0.0

        duration = (self.times[end_idx] - self.times[start_idx]).total_seconds()
        return max(duration + self.frame_period_seconds, self.frame_period_seconds)

    def generate_care_records(self):
        # Turning-care records are generated by turning_care_algorithm.py via
        # calculate_analysis(). Keep this method only for the existing UI call
        # sequence and older integration code.
        return

    def format_care_records(self) -> str:
        if not self.care_records:
            return "無照護紀錄"

        total_seconds = sum(record.duration_seconds for record in self.care_records)
        lines = [
            f"照護次數: {len(self.care_records)}",
            f"總照護時間: {total_seconds:.1f} 秒 ({total_seconds / 60:.2f} 分鐘)",
            "",
        ]

        for index, record in enumerate(self.care_records, start=1):
            lines.extend(
                [
                    f"照護 {index}",
                    f"開始: {record.start_time}",
                    f"結束: {record.end_time}",
                    f"持續: {record.duration_seconds:.1f} 秒 ({record.duration_seconds / 60:.2f} 分鐘)",
                    f"Frame: {record.start_index} - {record.end_index}",
                    "",
                ]
            )

        return "\n".join(lines)

    def care_record_summary(self) -> str:
        turning_seconds = sum(record.duration_seconds for record in self.care_records)
        edge_seconds = sum(record.duration_seconds for record in self.bed_edge_contact_records)
        return (
            f"翻身照護: {len(self.care_records)} 次 / {turning_seconds:.1f} 秒 | "
            f"床緣接觸: {len(self.bed_edge_contact_records)} 次 / {edge_seconds:.1f} 秒"
        )

    def export_backend_events(self) -> list[dict]:
        # Backend integration:
        # Use this as the unified source for database sync.
        # It returns the same two event categories shown in the lower-left list:
        # - edge_alert: patient touched the marked bed edge
        # - turning_care_event: turning-care event
        events: list[dict] = []
        for index, record in enumerate(self.care_records):
            raw_start, raw_end = self.care_record_event_ranges[index]
            events.append(
                {
                    "event_type": "turning_care_event",
                    "start_time": record.start_time,
                    "end_time": record.end_time,
                    "duration_seconds": record.duration_seconds,
                    "start_index": record.start_index,
                    "end_index": record.end_index,
                    "waveform_start_index": raw_start,
                    "waveform_end_index": raw_end,
                    "activity_score": record.activity_score,
                }
            )

        for record in self.bed_edge_contact_records:
            events.append(
                {
                    "event_type": "edge_alert",
                    "start_time": record.start_time,
                    "end_time": record.end_time,
                    "duration_seconds": record.duration_seconds,
                    "start_index": record.start_index,
                    "end_index": record.end_index,
                }
            )

        return events

    def populate_care_record_list(self):
        # This method renders the lower-left event list from two record sources:
        # 1. self.care_records -> turning-care events ("turning_care_event")
        # 2. self.bed_edge_contact_records -> edge-alert events ("edge_alert")
        # self.event_list_items only maps visible list rows back to record indexes for UI clicks;
        # it is not the canonical event data source.
        self.record_summary_label.setText(self.care_record_summary())
        self.record_list.clear()
        self.event_list_items = []

        if not self.care_records and not self.bed_edge_contact_records:
            item = QListWidgetItem("無事件紀錄")
            item.setFlags(item.flags() & ~Qt.ItemIsSelectable)
            self.record_list.addItem(item)
            return

        for index, record in enumerate(self.care_records):
            raw_start, raw_end = self.care_record_event_ranges[index]
            motion_seconds = self.motion_duration_seconds(raw_start, raw_end)
            item = QListWidgetItem(
                f"{record.activity_type} {index + 1} | {record.duration_seconds:.1f} 秒 | "
                f"motion {motion_seconds:.1f} 秒\n"
                f"{record.start_time} - {record.end_time}\n"
                f"Waveform frame: {raw_start} - {raw_end} | turn score: {record.activity_score}/4"
            )
            item.setData(Qt.UserRole, index)
            self.record_list.addItem(item)
            self.event_list_items.append(("turning", index))

        for index, record in enumerate(self.bed_edge_contact_records):
            item = QListWidgetItem(
                f"床緣接觸 {index + 1} | {record.duration_seconds:.1f} 秒\n"
                f"{record.start_time} - {record.end_time}\n"
                f"Frame: {record.start_index} - {record.end_index}"
            )
            item.setData(Qt.UserRole, index)
            self.record_list.addItem(item)
            self.event_list_items.append(("bed_edge", index))

    def focus_care_record(self, row: int):
        if row < 0 or row >= len(self.event_list_items):
            return

        event_type, event_index = self.event_list_items[row]
        if event_type == "turning":
            start_idx, end_idx = self.care_record_event_ranges[event_index]
        else:
            record = self.bed_edge_contact_records[event_index]
            start_idx, end_idx = record.start_index, record.end_index

        width = max(10.0, float(end_idx - start_idx + 1))
        margin = max(5.0, width * 0.35)
        self.set_chart_xlim(start_idx - margin, end_idx + margin)
        self.slider.setValue(start_idx)

    def is_care_frame(self, idx: int) -> bool:
        return any(
            record.start_index <= idx <= record.end_index for record in self.care_records
        )

    def draw_analysis_chart(self):
        current_xlim = self.ax_chart.get_xlim() if self.ax_chart.has_data() else None
        self.ax_chart.clear()
        self.cursor_line = None
        step = max(1, len(self.care_scores) // 2000)
        x = np.arange(0, len(self.care_scores), step)

        self.ax_chart.plot(
            x,
            self.care_scores[::step],
            label=(
                "Raw Care Score"
                if self.disable_score_smoothing
                else f"Smoothed Care Score ({self.score_smooth_seconds:.1f}s)"
            ),
            color=CYAN,
            linewidth=1.4,
        )
        motion_threshold = self.get_motion_threshold_score()
        self.ax_chart.axhline(
            motion_threshold,
            linestyle="--",
            color=MAGENTA,
            linewidth=1.3,
            alpha=0.95,
            label=f"Motion Threshold ({motion_threshold:.2f})",
        )
        chart_top = float(max(np.max(self.care_scores), motion_threshold, 1))
        waveform_labeled = False
        turn_labeled = False
        for event_index, (start_idx, end_idx) in enumerate(self.care_record_event_ranges):
            record = self.care_records[event_index] if event_index < len(self.care_records) else None
            is_turning = record is not None and record.activity_type == "翻身照護"
            event_color = BLUE if is_turning else ORANGE
            event_label = "翻身照護" if is_turning else "Qualified Motion"
            should_label = (is_turning and not turn_labeled) or (not is_turning and not waveform_labeled)
            segment_x = np.arange(start_idx, end_idx + 1)
            self.ax_chart.axvspan(
                start_idx,
                end_idx,
                alpha=0.22 if is_turning else 0.18,
                color=event_color,
            )
            self.ax_chart.plot(
                segment_x,
                self.care_scores[start_idx : end_idx + 1],
                color=event_color,
                linewidth=2.4,
                label=event_label if should_label else None,
            )
            if is_turning:
                turn_labeled = True
            else:
                waveform_labeled = True
            self.ax_chart.axvline(start_idx, color=GREEN, linestyle="--", alpha=0.9)
            self.ax_chart.axvline(end_idx, color=RED, linestyle="--", alpha=0.9)
            self.ax_chart.text(
                start_idx,
                chart_top * 0.96,
                "Start",
                fontsize=8,
                color=GREEN,
                rotation=90,
            )
            self.ax_chart.text(
                end_idx,
                chart_top * 0.96,
                "End",
                fontsize=8,
                color=RED,
                rotation=90,
            )

        for record in self.care_records:
            self.ax_chart.axvspan(
                record.start_index,
                record.end_index,
                alpha=0.08,
                color=ORANGE,
            )

        edge_labeled = False
        for record in self.bed_edge_contact_records:
            self.ax_chart.axvspan(
                record.start_index,
                record.end_index,
                alpha=0.20,
                color=VIOLET,
                label="床緣接觸" if not edge_labeled else None,
            )
            edge_labeled = True
            self.ax_chart.axvline(record.start_index, color=VIOLET, linestyle="--", alpha=0.9)
            self.ax_chart.axvline(record.end_index, color=VIOLET, linestyle="--", alpha=0.9)
            self.ax_chart.text(
                record.start_index,
                chart_top * 0.82,
                "Edge Start",
                fontsize=8,
                color=VIOLET,
                rotation=90,
            )
            self.ax_chart.text(
                record.end_index,
                chart_top * 0.82,
                "Edge End",
                fontsize=8,
                color=VIOLET,
                rotation=90,
            )

        style_axis(self.ax_chart, "Nursing Care Analysis", "Frame", "Care Score")
        legend = self.ax_chart.legend(loc="upper right")
        legend.get_frame().set_facecolor("#0b1220")
        legend.get_frame().set_edgecolor("#21436f")
        for text in legend.get_texts():
            text.set_color(TEXT_COLOR)
        self.ax_chart.grid(True, color=GRID_COLOR, alpha=0.5)
        if current_xlim is not None:
            self.ax_chart.set_xlim(current_xlim)

    def update_frame(self, idx):
        frame = self.frames[idx]
        max_temp = float(np.max(frame))
        min_temp = float(np.min(frame))

        self.im.set_data(frame)
        self.im.set_clim(vmin=min_temp, vmax=max_temp)
        self.update_bed_edge_contact_alert(frame)

        self.clear_cursor_line()
        self.cursor_line = self.ax_chart.axvline(idx, color="#ffffff", linestyle="--", linewidth=1)
        self.canvas.draw_idle()

    def clear_cursor_line(self):
        if self.cursor_line is None:
            return

        try:
            self.cursor_line.remove()
        except (NotImplementedError, ValueError):
            pass
        self.cursor_line = None

    def slider_changed(self, value):
        self.index = value
        self.update_frame(value)

    def next_frame(self):
        if self.index < len(self.frames) - 1:
            self.index += 1
            self.slider.setValue(self.index)
        else:
            self.timer.stop()

    def prev_manual(self):
        if self.index > 0:
            self.index -= 1
            self.slider.setValue(self.index)

    def next_manual(self):
        if self.index < len(self.frames) - 1:
            self.index += 1
            self.slider.setValue(self.index)

    def play(self):
        self.timer.start(100)

    def stop(self):
        self.timer.stop()


class AnalysisUI(QWidget):
    def __init__(self, pg_config: PostgresConfig, table_name: str):
        super().__init__()
        self.pg_config = pg_config
        self.table_name = table_name
        self.windows = []

        self.setWindowTitle("熱成像與護理師照護分析工具")
        self.resize(500, 500)
        self.setStyleSheet(APP_STYLE)

        layout = QVBoxLayout()

        layout.addWidget(QLabel("Host"))
        self.host_input = QLineEdit(pg_config.host)
        layout.addWidget(self.host_input)

        layout.addWidget(QLabel("Port"))
        self.port_input = QLineEdit(str(pg_config.port))
        layout.addWidget(self.port_input)

        layout.addWidget(QLabel("Username"))
        self.username_input = QLineEdit(pg_config.username)
        layout.addWidget(self.username_input)

        layout.addWidget(QLabel("Password"))
        self.password_input = QLineEdit(pg_config.password)
        self.password_input.setEchoMode(QLineEdit.Password)
        layout.addWidget(self.password_input)

        layout.addWidget(QLabel("Database"))
        self.database_input = QLineEdit(pg_config.database)
        layout.addWidget(self.database_input)

        layout.addWidget(QLabel("Table"))
        self.table_input = QLineEdit(table_name)
        layout.addWidget(self.table_input)

        layout.addWidget(QLabel("Sensor"))
        self.sensor_combo = QComboBox()
        self.sensor_combo.addItem("尚未載入 sensors", None)
        layout.addWidget(self.sensor_combo)

        reload_sensors_btn = QPushButton("重新載入 Sensor 清單")
        reload_sensors_btn.clicked.connect(self.reload_sensor_options)
        layout.addWidget(reload_sensors_btn)

        start_datetime, end_datetime = today_datetime_range()

        layout.addWidget(QLabel("開始時間"))
        self.start_input = DateTimePicker(start_datetime)
        layout.addWidget(self.start_input)

        layout.addWidget(QLabel("結束時間"))
        self.end_input = DateTimePicker(end_datetime)
        layout.addWidget(self.end_input)

        btn = QPushButton("讀取熱成像並分析照護時間")
        btn.clicked.connect(self.generate_heatmap)
        layout.addWidget(btn)

        self.setLayout(layout)
        QTimer.singleShot(0, lambda: self.reload_sensor_options(show_errors=False))

    def show_loading_mask(self, message: str) -> QProgressDialog:
        loading = QProgressDialog(message, "", 0, 0, self)
        loading.setWindowTitle("請稍候")
        loading.setWindowModality(Qt.ApplicationModal)
        loading.setCancelButton(None)
        loading.setMinimumDuration(0)
        loading.setAutoClose(False)
        loading.setAutoReset(False)
        loading.setStyleSheet(APP_STYLE)
        loading.show()
        QApplication.setOverrideCursor(Qt.WaitCursor)
        QApplication.processEvents()
        return loading

    def close_loading_mask(self, loading: QProgressDialog):
        QApplication.restoreOverrideCursor()
        loading.close()
        loading.deleteLater()
        QApplication.processEvents()

    def collect_pg_config(self) -> PostgresConfig:
        return PostgresConfig(
            host=self.host_input.text().strip() or DEFAULT_PG_HOST,
            port=int(self.port_input.text().strip() or DEFAULT_PG_PORT),
            username=self.username_input.text().strip() or DEFAULT_PG_USER,
            password=self.password_input.text(),
            database=self.database_input.text().strip() or DEFAULT_PG_DATABASE,
        )

    def reload_sensor_options(self, show_errors: bool = True):
        try:
            pg_config = self.collect_pg_config()
            sensor_options = load_sensor_options(pg_config)
        except Exception as exc:
            self.sensor_combo.clear()
            self.sensor_combo.addItem("載入 sensors 失敗", None)
            if show_errors:
                QMessageBox.critical(self, "資料庫錯誤", str(exc))
            return

        self.sensor_combo.clear()
        if not sensor_options:
            self.sensor_combo.addItem("sensors 無資料", None)
            return

        default_index = 0
        for index, sensor in enumerate(sensor_options):
            self.sensor_combo.addItem(sensor.sensor_number, sensor.sensor_id)
            if str(sensor.sensor_id) == DEFAULT_SENSOR_ID:
                default_index = index
        self.sensor_combo.setCurrentIndex(default_index)

    def generate_heatmap(self):
        try:
            pg_config = self.collect_pg_config()
            table_name = self.table_input.text().strip() or DEFAULT_TABLE
            sensor_id = self.sensor_combo.currentData()
            sensor_number = self.sensor_combo.currentText()
            if sensor_id is None:
                raise ValueError("請先載入並選擇 Sensor。")
            sensor_id = int(sensor_id)
            start = pd.Timestamp(self.start_input.dateTime().toPyDateTime())
            end = pd.Timestamp(self.end_input.dateTime().toPyDateTime())
            if start >= end:
                raise ValueError("開始時間必須早於結束時間")
        except Exception as exc:
            QMessageBox.warning(self, "輸入錯誤", str(exc))
            return

        loading = self.show_loading_mask("正在讀取熱成像資料並分析，請稍候...")
        try:
            try:
                frames, times, skipped = load_frames_from_postgres(
                    config=pg_config,
                    table_name=table_name,
                    sensor_id=sensor_id,
                    start=start,
                    end=end,
                )
            except Exception as exc:
                self.close_loading_mask(loading)
                QMessageBox.critical(self, "資料庫錯誤", str(exc))
                return

            if not frames:
                self.close_loading_mask(loading)
                QMessageBox.information(
                    self,
                    "沒有資料",
                    f"查無可用熱成像資料。略過筆數: {skipped}",
                )
                return

            title = f"Sensor {sensor_number} Thermal - {pg_config.host}:{pg_config.port}/{pg_config.database}"
            window = HeatmapWindow(
                frames,
                times,
                title,
                pg_config=pg_config,
                sensor_id=sensor_id,
                sensor_number=sensor_number,
            )
            self.windows.append(window)
            window.destroyed.connect(lambda: self.windows.remove(window))
            window.show()
            self.lower()
            window.raise_()
            window.activateWindow()
        finally:
            if loading.isVisible():
                self.close_loading_mask(loading)

        if skipped:
            QMessageBox.information(self, "提醒", f"已略過 {skipped} 筆無法解析的資料。")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Analyze PostgreSQL thermal frame data and nursing care time."
    )
    parser.add_argument("--host", default=DEFAULT_PG_HOST, help="PostgreSQL host.")
    parser.add_argument("--port", type=int, default=DEFAULT_PG_PORT, help="PostgreSQL port.")
    parser.add_argument("--user", default=DEFAULT_PG_USER, help="PostgreSQL username.")
    parser.add_argument(
        "--password",
        default=DEFAULT_PG_PASSWORD,
        help="PostgreSQL password.",
    )
    parser.add_argument(
        "--database",
        default=DEFAULT_PG_DATABASE,
        help="PostgreSQL database name.",
    )
    parser.add_argument("--table", default=DEFAULT_TABLE, help="PostgreSQL table name.")
    return parser.parse_args()


if __name__ == "__main__":
    args = parse_args()
    configure_matplotlib_fonts()
    app = QApplication(sys.argv)
    ui = AnalysisUI(
        pg_config=PostgresConfig(
            host=args.host,
            port=args.port,
            username=args.user,
            password=args.password,
            database=args.database,
        ),
        table_name=args.table,
    )
    ui.show()
    sys.exit(app.exec_())
