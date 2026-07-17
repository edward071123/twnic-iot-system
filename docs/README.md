# CARE SENSE 產品簡報

- `care_sense_product_overview_zh-TW.pptx`：可直接編輯的 16:9 商業簡報，共 15 頁，包含雙感測模組、產品三視角與安裝介面介紹。
- `care_sense_product_overview_preview.png`：全頁縮圖預覽。
- `care_sense_product_deck_source/`：PptxGenJS 原始碼與版面輔助程式；所有視覺均為可編輯的抽象示意圖，不含真人熱成像素材。
- `refences/transparent/`：三張產品照片的透明背景 PNG，供簡報與其他商業素材使用。

## 重新產生簡報

```bash
cd care_sense_product_deck_source
npm install
npm run generate
```

輸出檔案為 `care_sense_product_overview_zh-TW.pptx`。

## 內容定位

簡報以醫療照護場域的商業溝通為主，採米白、深咖啡與珊瑚紅色系，聚焦無接觸量測、減少逐床量測勞力、AIoT 即時警報，以及 API／報表串接能力。體表溫度與其他指標定位為照護輔助資訊，不取代醫療診斷。
