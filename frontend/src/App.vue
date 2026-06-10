<script lang="ts" setup>
import Main from './components/Main.vue';
import { NMessageProvider, NDialogProvider } from 'naive-ui';
</script>

<template>
  <!-- <img id="logo" alt="Wails logo" src="./assets/images/logo-universal.png"/> -->
  <n-message-provider>
    <n-dialog-provider>
      <Main />
    </n-dialog-provider>
  </n-message-provider>
</template>

<style>
#logo {
  display: block;
  width: 50%;
  height: 50%;
  margin: auto;
  padding: 10% 0 0;
  background-position: center;
  background-repeat: no-repeat;
  background-size: 100% 100%;
  background-origin: content-box;
}
</style>
