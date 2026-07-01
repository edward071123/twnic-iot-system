"""Turning-care detection protocol.

This module is intentionally independent from the PyQt UI and database.  The
main program passes thermal frames and timestamps in, then receives turning-care
event records out.  Bed-edge points are sensor configuration used by the caller
for edge-contact detection; they are not part of this turning-care classifier.
Future algorithm changes should be made here first.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Sequence

import numpy as np


TURNING_CARE_ACTIVITY_TYPE = "\u7ffb\u8eab\u7167\u8b77"
NON_TURNING_CARE_ACTIVITY_TYPE = "\u975e\u7ffb\u8eab\u7167\u8b77"
CARE_ACTIVITY_TYPE = "\u7167\u8b77"


@dataclass(frozen=True)
class TurningCareConfig:
    """Tunable thresholds for turning-care detection."""

    min_care_motion_seconds: float = 5.0
    care_event_gap_seconds: float = 0.75
    care_record_merge_gap_seconds: float = 8.0
    care_record_padding_seconds: float = 3.0
    default_motion_threshold_score: float = 3.0
    default_score_smooth_seconds: float = 60.0
    wave_base_ratio: float = 0.18
    turn_centroid_displacement_min: float = 1.8
    turn_changed_area_min: float = 0.075
    turn_changed_width_min: float = 22.0
    turn_changed_height_min: float = 18.0
    turn_temp_mean_delta_min: float = 0.65
    reposition_changed_area_min: float = 0.075
    reposition_changed_width_min: float = 22.0
    reposition_changed_height_min: float = 18.0
    reposition_centroid_path_min: float = 35.0
    reposition_warm_area_min: float = 0.20
    reposition_cool_area_min: float = 0.08
    turning_care_merge_gap_seconds: float = 600.0


@dataclass
class TurningCareRecord:
    start_index: int
    end_index: int
    start_time: object
    end_time: object
    frame_period_seconds: float
    activity_type: str = TURNING_CARE_ACTIVITY_TYPE
    activity_score: int = 0

    @property
    def duration_seconds(self) -> float:
        duration = (self.end_time - self.start_time).total_seconds()
        return max(duration, self.frame_period_seconds)


@dataclass(frozen=True)
class TurningCareAnalysis:
    motion_scores: np.ndarray
    care_scores: np.ndarray
    care_indices: set[int]
    care_event_ranges: list[tuple[int, int]]
    care_record_event_ranges: list[tuple[int, int]]
    care_records: list[TurningCareRecord]


def smooth_signal(data: Sequence[float], window: int = 50) -> np.ndarray:
    data = np.asarray(data, dtype=float)
    if len(data) < window or window <= 1:
        return data

    kernel = np.ones(window) / window
    return np.convolve(data, kernel, mode="same")


def robust_scale(data: Sequence[float]) -> np.ndarray:
    data = np.asarray(data, dtype=float)
    if data.size == 0:
        return data

    median = np.median(data)
    mad = np.median(np.abs(data - median))
    if mad == 0:
        std = np.std(data)
        if std == 0:
            return np.zeros_like(data)
        return np.clip((data - np.mean(data)) / std, 0, None)
    return np.clip((data - median) / (1.4826 * mad), 0, None)


def weighted_hot_centroid(frame: np.ndarray) -> np.ndarray | None:
    threshold = np.percentile(frame, 80)
    weights = np.clip(frame - threshold, 0, None)
    total = float(weights.sum())
    if total <= 0:
        return None

    y_indices, x_indices = np.indices(frame.shape)
    x = float((x_indices * weights).sum() / total)
    y = float((y_indices * weights).sum() / total)
    return np.array([x, y], dtype=float)


def analyze_turning_care_events(
    frames: Sequence[np.ndarray],
    times: Sequence[object],
    *,
    frame_period_seconds: float,
    motion_threshold_score: float | None = None,
    score_smooth_seconds: float | None = None,
    disable_score_smoothing: bool = False,
    config: TurningCareConfig | None = None,
) -> TurningCareAnalysis:
    """Analyze thermal frames and return turning-care events.

    Protocol:
    - `frames`: ordered 2D thermal arrays. Each frame must have the same shape.
    - `times`: ordered timestamp objects matching `frames`; each object must
      support timestamp subtraction with `.total_seconds()`.
    - output records are already filtered to turning-care events only.
    """

    config = config or TurningCareConfig()
    motion_threshold_score = (
        config.default_motion_threshold_score
        if motion_threshold_score is None
        else float(motion_threshold_score)
    )
    score_smooth_seconds = (
        config.default_score_smooth_seconds
        if score_smooth_seconds is None
        else float(score_smooth_seconds)
    )

    motion_scores, care_scores = calculate_scores(
        frames,
        frame_period_seconds=frame_period_seconds,
        score_smooth_seconds=score_smooth_seconds,
        disable_score_smoothing=disable_score_smoothing,
    )
    raw_care_indices = {
        i for i, score in enumerate(care_scores) if score > motion_threshold_score
    }
    care_indices, care_event_ranges = filter_persistent_indices(
        raw_care_indices,
        care_scores,
        times,
        frame_period_seconds=frame_period_seconds,
        config=config,
    )
    care_records, care_record_event_ranges = generate_care_records(
        frames,
        times,
        care_indices,
        frame_period_seconds=frame_period_seconds,
        config=config,
    )
    return TurningCareAnalysis(
        motion_scores=motion_scores,
        care_scores=care_scores,
        care_indices=care_indices,
        care_event_ranges=care_event_ranges,
        care_record_event_ranges=care_record_event_ranges,
        care_records=care_records,
    )


def calculate_scores(
    frames: Sequence[np.ndarray],
    *,
    frame_period_seconds: float,
    score_smooth_seconds: float,
    disable_score_smoothing: bool,
) -> tuple[np.ndarray, np.ndarray]:
    mean_motion_scores = []
    p95_motion_scores = []
    centroid_shift_scores = []
    max_delta_scores = []
    prev_frame = None
    prev_centroid = None

    for frame in frames:
        if prev_frame is None:
            mean_motion_scores.append(0.0)
            p95_motion_scores.append(0.0)
            max_delta_scores.append(0.0)
        else:
            diff = np.abs(frame - prev_frame)
            mean_motion_scores.append(float(np.mean(diff)))
            p95_motion_scores.append(float(np.percentile(diff, 95)))
            max_delta_scores.append(float(abs(np.max(frame) - np.max(prev_frame))))

        centroid = weighted_hot_centroid(frame)
        if centroid is None or prev_centroid is None:
            centroid_shift_scores.append(0.0)
        else:
            centroid_shift_scores.append(float(np.linalg.norm(centroid - prev_centroid)))
        if centroid is not None:
            prev_centroid = centroid
        prev_frame = frame

    mean_motion = smooth_signal(mean_motion_scores, window=32)
    p95_motion = smooth_signal(p95_motion_scores, window=32)
    centroid_shift = smooth_signal(centroid_shift_scores, window=32)
    max_delta = smooth_signal(max_delta_scores, window=32)

    smooth_window = score_smooth_window(
        frame_period_seconds,
        score_smooth_seconds,
        disable_score_smoothing=disable_score_smoothing,
    )
    care_scores = smooth_signal(
        robust_scale(mean_motion) * 0.48
        + robust_scale(p95_motion) * 0.31
        + robust_scale(centroid_shift) * 0.15
        + robust_scale(max_delta) * 0.06,
        window=smooth_window,
    )
    return mean_motion, care_scores


def score_smooth_window(
    frame_period_seconds: float,
    score_smooth_seconds: float,
    *,
    disable_score_smoothing: bool,
) -> int:
    if disable_score_smoothing:
        return 1
    return max(1, int(round(score_smooth_seconds / max(frame_period_seconds, 0.1))))


def filter_persistent_indices(
    indices: set[int],
    care_scores: np.ndarray,
    times: Sequence[object],
    *,
    frame_period_seconds: float,
    config: TurningCareConfig,
) -> tuple[set[int], list[tuple[int, int]]]:
    event_ranges: list[tuple[int, int]] = []
    if not indices:
        return set(), event_ranges

    sorted_indices = sorted(indices)
    merge_gap_frames = max(
        1,
        int(round(config.care_event_gap_seconds / max(frame_period_seconds, 0.1))),
    )
    filtered: set[int] = set()
    start_idx = sorted_indices[0]
    prev_idx = sorted_indices[0]

    for idx in sorted_indices[1:]:
        if idx - prev_idx <= merge_gap_frames:
            prev_idx = idx
            continue

        if motion_duration_seconds(times, frame_period_seconds, start_idx, prev_idx) >= config.min_care_motion_seconds:
            wave_start, wave_end = expand_to_wave_base(
                care_scores,
                frame_period_seconds=frame_period_seconds,
                start_idx=start_idx,
                end_idx=prev_idx,
                config=config,
            )
            event_ranges.append((wave_start, wave_end))
            filtered.update(range(wave_start, wave_end + 1))
        start_idx = idx
        prev_idx = idx

    if motion_duration_seconds(times, frame_period_seconds, start_idx, prev_idx) >= config.min_care_motion_seconds:
        wave_start, wave_end = expand_to_wave_base(
            care_scores,
            frame_period_seconds=frame_period_seconds,
            start_idx=start_idx,
            end_idx=prev_idx,
            config=config,
        )
        event_ranges.append((wave_start, wave_end))
        filtered.update(range(wave_start, wave_end + 1))
    return filtered, event_ranges


def expand_to_wave_base(
    care_scores: np.ndarray,
    *,
    frame_period_seconds: float,
    start_idx: int,
    end_idx: int,
    config: TurningCareConfig,
) -> tuple[int, int]:
    if care_scores.size == 0:
        return start_idx, end_idx

    max_expand_frames = max(10, int(round(20.0 / max(frame_period_seconds, 0.1))))
    local_start = max(0, start_idx - max_expand_frames)
    local_end = min(len(care_scores) - 1, end_idx + max_expand_frames)
    local_scores = care_scores[local_start : local_end + 1]
    peak = float(np.max(care_scores[start_idx : end_idx + 1]))
    baseline = float(np.percentile(local_scores, 20))
    cutoff = baseline + (peak - baseline) * config.wave_base_ratio

    wave_start = start_idx
    min_start = max(0, start_idx - max_expand_frames)
    while wave_start > min_start and care_scores[wave_start] > cutoff:
        wave_start -= 1

    wave_end = end_idx
    max_end = min(len(care_scores) - 1, end_idx + max_expand_frames)
    while wave_end < max_end and care_scores[wave_end] > cutoff:
        wave_end += 1

    return wave_start, wave_end


def motion_duration_seconds(
    times: Sequence[object],
    frame_period_seconds: float,
    start_idx: int,
    end_idx: int,
) -> float:
    if len(times) == 0:
        return 0.0

    duration = (times[end_idx] - times[start_idx]).total_seconds()
    return max(duration + frame_period_seconds, frame_period_seconds)


def generate_care_records(
    frames: Sequence[np.ndarray],
    times: Sequence[object],
    care_indices: set[int],
    *,
    frame_period_seconds: float,
    config: TurningCareConfig,
) -> tuple[list[TurningCareRecord], list[tuple[int, int]]]:
    records: list[TurningCareRecord] = []
    record_event_ranges: list[tuple[int, int]] = []
    if not care_indices:
        return records, record_event_ranges

    sorted_indices = sorted(care_indices)
    merge_gap_frames = max(
        3,
        int(round(config.care_record_merge_gap_seconds / max(frame_period_seconds, 0.1))),
    )
    start_idx = sorted_indices[0]
    prev_idx = sorted_indices[0]

    for idx in sorted_indices[1:]:
        if idx - prev_idx <= merge_gap_frames:
            prev_idx = idx
            continue

        append_care_record(
            records,
            record_event_ranges,
            frames,
            times,
            start_idx,
            prev_idx,
            frame_period_seconds=frame_period_seconds,
            config=config,
        )
        start_idx = idx
        prev_idx = idx

    append_care_record(
        records,
        record_event_ranges,
        frames,
        times,
        start_idx,
        prev_idx,
        frame_period_seconds=frame_period_seconds,
        config=config,
    )
    return records, record_event_ranges


def append_care_record(
    records: list[TurningCareRecord],
    record_event_ranges: list[tuple[int, int]],
    frames: Sequence[np.ndarray],
    times: Sequence[object],
    start_idx: int,
    end_idx: int,
    *,
    frame_period_seconds: float,
    config: TurningCareConfig,
) -> None:
    if motion_duration_seconds(times, frame_period_seconds, start_idx, end_idx) < config.min_care_motion_seconds:
        return

    padding_frames = max(
        1,
        int(round(config.care_record_padding_seconds / max(frame_period_seconds, 0.1))),
    )
    padded_start = max(0, start_idx - padding_frames)
    padded_end = min(len(frames) - 1, end_idx + padding_frames)
    activity_type, activity_score = classify_care_activity(
        frames,
        start_idx,
        end_idx,
        frame_period_seconds=frame_period_seconds,
        config=config,
    )
    if activity_type != TURNING_CARE_ACTIVITY_TYPE:
        return

    if merge_turning_care_record(
        records,
        record_event_ranges,
        times,
        start_idx,
        end_idx,
        padded_start,
        padded_end,
        activity_score,
        config=config,
    ):
        return

    record_event_ranges.append((start_idx, end_idx))
    records.append(
        TurningCareRecord(
            start_index=padded_start,
            end_index=padded_end,
            start_time=times[padded_start],
            end_time=times[padded_end],
            frame_period_seconds=frame_period_seconds,
            activity_type=activity_type,
            activity_score=activity_score,
        )
    )


def merge_turning_care_record(
    records: list[TurningCareRecord],
    record_event_ranges: list[tuple[int, int]],
    times: Sequence[object],
    start_idx: int,
    end_idx: int,
    padded_start: int,
    padded_end: int,
    activity_score: int,
    *,
    config: TurningCareConfig,
) -> bool:
    if not records:
        return False

    last_raw_start, last_raw_end = record_event_ranges[-1]
    gap_seconds = (times[start_idx] - times[last_raw_end]).total_seconds()
    if gap_seconds > config.turning_care_merge_gap_seconds:
        return False

    last_record = records[-1]
    record_event_ranges[-1] = (last_raw_start, end_idx)
    last_record.end_index = max(last_record.end_index, padded_end)
    last_record.end_time = times[last_record.end_index]
    last_record.activity_score = max(last_record.activity_score, activity_score)
    last_record.start_index = min(last_record.start_index, padded_start)
    last_record.start_time = times[last_record.start_index]
    return True


def classify_care_activity(
    frames: Sequence[np.ndarray],
    start_idx: int,
    end_idx: int,
    *,
    frame_period_seconds: float,
    config: TurningCareConfig,
) -> tuple[str, int]:
    event_frames = frames[start_idx : end_idx + 1]
    if not event_frames:
        return CARE_ACTIVITY_TYPE, 0

    context_frames = max(30, int(round(300.0 / max(frame_period_seconds, 0.1))))
    before = frames[max(0, start_idx - context_frames) : start_idx]
    after = frames[end_idx + 1 : min(len(frames), end_idx + context_frames + 1)]
    background_frames = list(before) + list(after)
    if background_frames:
        background = np.median(np.stack(background_frames), axis=0)
    else:
        background = np.median(np.stack(event_frames), axis=0)

    centroids = []
    changed_areas = []
    changed_widths = []
    changed_heights = []
    temp_mean_deltas = []
    warm_areas = []
    cool_areas = []
    for frame in event_frames:
        centroid = weighted_hot_centroid(frame)
        if centroid is not None:
            centroids.append(centroid)

        delta = np.abs(frame - background)
        changed_threshold = max(0.8, float(np.percentile(delta, 92)))
        changed = delta >= changed_threshold
        changed_areas.append(float(np.mean(changed)))

        ys, xs = np.where(changed)
        changed_widths.append(float(xs.max() - xs.min() + 1) if xs.size else 0.0)
        changed_heights.append(float(ys.max() - ys.min() + 1) if ys.size else 0.0)
        signed_delta = frame - background
        temp_mean_deltas.append(float(np.mean(signed_delta)))
        warm_areas.append(float(np.mean(signed_delta > 0.8)))
        cool_areas.append(float(np.mean(signed_delta < -0.8)))

    centroid_displacement = 0.0
    centroid_path = 0.0
    if len(centroids) > 1:
        centroid_array = np.asarray(centroids, dtype=float)
        centroid_displacement = float(np.linalg.norm(centroid_array[-1] - centroid_array[0]))
        centroid_path = float(
            np.sum(np.linalg.norm(np.diff(centroid_array, axis=0), axis=1))
        )

    changed_area = float(np.mean(changed_areas)) if changed_areas else 0.0
    changed_width = float(np.mean(changed_widths)) if changed_widths else 0.0
    changed_height = float(np.mean(changed_heights)) if changed_heights else 0.0
    temp_mean_delta = float(np.mean(temp_mean_deltas)) if temp_mean_deltas else 0.0
    warm_area = float(np.mean(warm_areas)) if warm_areas else 0.0
    cool_area = float(np.mean(cool_areas)) if cool_areas else 0.0

    turn_score = 0
    if centroid_displacement >= config.turn_centroid_displacement_min:
        turn_score += 1
    if changed_area >= config.turn_changed_area_min:
        turn_score += 1
    if changed_width >= config.turn_changed_width_min:
        turn_score += 1
    if temp_mean_delta >= config.turn_temp_mean_delta_min:
        turn_score += 1

    full_body_turn = changed_height >= config.turn_changed_height_min and turn_score >= 3
    heat_redistribution_turn = (
        changed_area >= config.reposition_changed_area_min
        and changed_width >= config.reposition_changed_width_min
        and changed_height >= config.reposition_changed_height_min
        and centroid_path >= config.reposition_centroid_path_min
        and warm_area >= config.reposition_warm_area_min
        and cool_area >= config.reposition_cool_area_min
    )

    if full_body_turn or heat_redistribution_turn:
        return TURNING_CARE_ACTIVITY_TYPE, turn_score
    return NON_TURNING_CARE_ACTIVITY_TYPE, turn_score
