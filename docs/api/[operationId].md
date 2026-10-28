---
aside: false
outline: false
title: LONG API
---

<script setup lang="ts">
import { useRoute } from 'vitepress'

const route = useRoute()

const operationId = route.data.params.operationId
</script>

<OAOperation :operationId="operationId">
  <template v-if="operationId === 'getFullSync'" #playground>
    <p>
      Download this streaming export to a file using curl or the Python client in the
      <a href="/catalog-sync#download-a-full-export">catalog sync guide</a>.
    </p>
  </template>
</OAOperation>
