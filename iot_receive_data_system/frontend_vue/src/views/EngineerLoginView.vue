<template>
  <div class="engineer-login">
    <div class="card">
      <h1>工程模式登入</h1>
      <p class="hint">請輸入工程模式帳號密碼</p>
      <form @submit.prevent="onSubmit">
        <label>
          帳號
          <input v-model.trim="username" type="text" autocomplete="username" required />
        </label>
        <label>
          密碼
          <input v-model="password" type="password" autocomplete="current-password" required />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button type="submit">登入工程模式</button>
      </form>
      <p class="back">
        <RouterLink to="/">回大眾版首頁</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { markEngineerAuthed } from '../auth/engineerAuth'

const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')

const engineerUser = String(import.meta.env.VITE_ENGINEER_USERNAME || '')
const engineerPass = String(import.meta.env.VITE_ENGINEER_PASSWORD || '')

function onSubmit() {
  if (!engineerUser || !engineerPass) {
    error.value = '工程模式帳密尚未設定，請先在 .env 設定 VITE_ENGINEER_USERNAME / VITE_ENGINEER_PASSWORD。'
    return
  }
  if (username.value === engineerUser && password.value === engineerPass) {
    markEngineerAuthed()
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/engineer'
    router.replace(redirect)
    return
  }
  error.value = '帳號或密碼錯誤'
}
</script>

<style scoped>
.engineer-login {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 20px;
  background: linear-gradient(160deg, #0f2238, #173a5f);
}

.card {
  width: min(420px, 100%);
  background: #f8fbff;
  border-radius: 14px;
  padding: 22px;
  border: 1px solid #bfd2e6;
  box-shadow: 0 12px 32px rgba(7, 26, 44, 0.28);
}

h1 {
  margin: 0;
  font-size: 22px;
  color: #143a62;
}

.hint {
  margin: 6px 0 0;
  color: #486884;
  font-size: 13px;
}

form {
  margin-top: 16px;
  display: grid;
  gap: 12px;
}

label {
  display: grid;
  gap: 6px;
  color: #1f3f5e;
  font-size: 14px;
}

input {
  border: 1px solid #afc6dc;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 14px;
  outline: none;
}

input:focus {
  border-color: #2f8df5;
  box-shadow: 0 0 0 2px rgba(47, 141, 245, 0.2);
}

button {
  border: 0;
  border-radius: 999px;
  background: #1d4d7f;
  color: #fff;
  padding: 10px 14px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}

button:hover {
  background: #163e67;
}

.error {
  margin: 0;
  color: #c22020;
  font-size: 13px;
}

.back {
  margin: 12px 0 0;
  font-size: 13px;
}

.back a {
  color: #1d4d7f;
}
</style>
