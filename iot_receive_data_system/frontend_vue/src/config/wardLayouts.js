export const wardFloorLayouts = {
  default: {
    rooms: [
      { base: '01', beds: 8, x: 790, y: 70, w: 170, h: 215, door: { x1: 868, y1: 285, x2: 882, y2: 285 }, label: { x: 853, y: 312 } },
      { base: '02', beds: 2, x: 865, y: 350, w: 95, h: 150, door: { x1: 906, y1: 350, x2: 920, y2: 350 }, label: { x: 896, y: 532 } },
      { base: '03', beds: 8, x: 620, y: 70, w: 170, h: 215, door: { x1: 698, y1: 285, x2: 712, y2: 285 }, label: { x: 683, y: 312 } },
      { base: '05', beds: 8, x: 450, y: 70, w: 170, h: 215, door: { x1: 528, y1: 285, x2: 542, y2: 285 }, label: { x: 513, y: 312 } },
      { base: '06', beds: 6, x: 110, y: 350, w: 240, h: 160, door: { x1: 350, y1: 422, x2: 350, y2: 438 }, label: { x: 366, y: 435, textAnchor: 'start' } },
      { base: '07', beds: 8, x: 280, y: 70, w: 170, h: 215, door: { x1: 358, y1: 285, x2: 372, y2: 285 }, label: { x: 343, y: 312 } },
      { base: '08', beds: 6, x: 110, y: 510, w: 240, h: 140, door: { x1: 350, y1: 572, x2: 350, y2: 588 }, label: { x: 366, y: 588, textAnchor: 'start' } },
      { base: '09', beds: 6, x: 110, y: 70, w: 170, h: 215, door: { x1: 188, y1: 285, x2: 202, y2: 285 }, label: { x: 173, y: 312 } },
      { base: '10', beds: 2, x: 110, y: 665, w: 240, h: 145, door: { x1: 223, y1: 665, x2: 237, y2: 665 }, label: { x: 366, y: 745, textAnchor: 'start' } }
    ]
  },
  3: {
    rooms: [
      { base: '01', beds: 8, x: 110, y: 70, w: 170, h: 215, door: { x1: 188, y1: 285, x2: 202, y2: 285 }, label: { x: 173, y: 312 } },
      { base: '02', beds: 2, x: 110, y: 665, w: 240, h: 145, door: { x1: 223, y1: 665, x2: 237, y2: 665 }, label: { x: 366, y: 745, textAnchor: 'start' } },
      { base: '03', beds: 8, x: 280, y: 70, w: 170, h: 215, door: { x1: 358, y1: 285, x2: 372, y2: 285 }, label: { x: 343, y: 312 } },
      { base: '05', beds: 8, x: 450, y: 70, w: 170, h: 215, door: { x1: 528, y1: 285, x2: 542, y2: 285 }, label: { x: 513, y: 312 } },
      { base: '06', beds: 6, x: 110, y: 350, w: 240, h: 160, door: { x1: 350, y1: 422, x2: 350, y2: 438 }, label: { x: 366, y: 435, textAnchor: 'start' } },
      { base: '07', beds: 8, x: 620, y: 70, w: 170, h: 215, door: { x1: 698, y1: 285, x2: 712, y2: 285 }, label: { x: 683, y: 312 } },
      { base: '08', beds: 6, x: 110, y: 510, w: 240, h: 140, door: { x1: 350, y1: 572, x2: 350, y2: 588 }, label: { x: 366, y: 588, textAnchor: 'start' } },
      { base: '09', beds: 6, x: 790, y: 70, w: 170, h: 215, door: { x1: 868, y1: 285, x2: 882, y2: 285 }, label: { x: 853, y: 312 } },
      { base: '10', beds: 2, x: 865, y: 350, w: 95, h: 150, door: { x1: 906, y1: 350, x2: 920, y2: 350 }, label: { x: 896, y: 532 } }
    ]
  }
}

export const baseBedNumberMap = {
  '01': [5, 1, 6, 2, 7, 3, 8, 4],
  '03': [5, 1, 6, 2, 7, 3, 8, 4],
  '05': [5, 1, 6, 2, 7, 3, 8, 4],
  '07': [5, 1, 6, 2, 7, 3, 8, 4],
  '09': [4, 1, 5, 2, 6, 3],
}

export const floorBedNumberMaps = {
  3: {},
}

export const baseRoomGridOverrides = {
  '06': { cols: 3, padX: 0 },
  '08': { cols: 3, padX: 0 },
}

export const floorRoomGridOverrides = {
  3: {
    '0306': { cols: 2, padX: 8 },
    '0308': { cols: 2, padX: 8 },
  },
}

export const getFloorConfig = (configs, floor) => configs[floor] || configs.default || {}

export function buildRoomDefs(floor) {
  const layout = wardFloorLayouts[floor] || wardFloorLayouts.default
  const floorPrefix = String(floor).padStart(2, '0')
  return layout.rooms.map((room) => ({
    ...room,
    door: { ...room.door },
    label: room.label ? { ...room.label } : null,
    id: `${floorPrefix}${room.base}`,
  }))
}
