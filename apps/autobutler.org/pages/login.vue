<template>
  <div class="login-page">
    <div class="login-container">
      <h1>Login</h1>
      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" required />
        </div>
        <div class="form-group">
          <label for="password">Password</label>
          <input id="password" v-model="password" type="password" required />
        </div>
        <button type="submit">Login</button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
const email = ref("");
const password = ref("");

const handleSubmit = () => {
  // TODO: Implement login logic
  console.log("Login attempted");
};
</script>

<style scoped>
.login-container {
  max-width: 400px;
  margin: 0 auto;
  padding: 2rem;
  background: #242424;
  border-radius: 8px;
  border: 1px solid #333;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

input {
  padding: 0.5rem;
  background: #1a1a1a;
  border: 1px solid #333;
  border-radius: 4px;
  color: #e0e0e0;
}

button {
  padding: 0.75rem;
  background: var(--primary-color);
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
}
</style>
