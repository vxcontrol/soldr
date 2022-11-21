<template>
  <div>
    <el-tabs tab-position="left" v-model="leftTab">
    </el-tabs>
  </div>
</template>

<script>
const name = "empty";

module.exports = {
  name,
  props: ["protoAPI", "hash", "module", "eventsAPI", "modulesAPI", "components", "viewMode"],
  data: () => ({
    leftTab: undefined
  })
};
</script>
