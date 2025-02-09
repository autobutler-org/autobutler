<template>
  <div class="home">
    <h1>Welcome to ExoFlow</h1>
    <p>
      ExoFlow is an abstract set of templates and automation that allows for packaging a single codebase to run on multiple clouds. The goal is to simplify multi-cloud, multi-region failovers and deployments, enabling seamless and reliable response.
    </p>
    <h2>Key Features</h2>
    <ul>
      <li>Write platform-agnostic code that translates to the host platform</li>
      <li>Define applications as networked microclouds using a single manifest</li>
      <li>Orchestrate deployments across selected providers (AWS, Azure, etc.)</li>
      <li>Automatic failover and redeployment to another host if a microcloud fails</li>
      <li>Proxy-based API translation for seamless integration with cloud services</li>
    </ul>
  </div>
</template>

<script>
export default {
  name: 'Home',
};
</script>

<style scoped>
.home {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
  background-color: #1a1a1a;
  color: #e0e0e0;
}

h1 {
  color: #4caf50;
}

h2 {
  color: #81c784;
}

ul {
  list-style-type: none;
  padding: 0;
}

li::before {
  content: "•";
  color: #4caf50;
  display: inline-block;
  width: 1em;
  margin-left: -1em;
}
</style>
