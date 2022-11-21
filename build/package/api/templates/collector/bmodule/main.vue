<template>
  <div>
    <el-tabs tab-position="left" v-model="leftTab">
      <el-tab-pane name="api" :label="locale[$i18n.locale]['api']" v-if="viewMode === 'agent'">
        <div class="layout-margin-xl limit-length">
          <el-input placeholder="Please input log line" v-model="logline">
            <el-button
              slot="append"
              icon="el-icon-s-promotion"
              class="layout-row-none"
              @click="sendLogLine"
            >{{ locale[$i18n.locale]['buttonSendLogLine'] }}
            </el-button>
          </el-input>
        </div>
        <ul>
          <li :key="line" v-for="line in lines">{{ line }}</li>
        </ul>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
const name = "collector";

module.exports = {
  name,
  props: ["protoAPI", "hash", "module", "eventsAPI", "modulesAPI", "components", "viewMode"],
  data: () => ({
    leftTab: undefined,
    connection: {},
    lines: [],
    logline: "",
    locale: {
      ru: {
        api: "VX API",
        buttonSendLogLine: "Отправить строку",
        connected: "подключен",
        connError: "Ошибка подключения к серверу",
        recvError: "Ошибка при выполнении"
      },
      en: {
        api: "VX API",
        buttonSendLogLine: "Send line",
        connected: "connected",
        connError: "Error connection to the server",
        recvError: "Error on execute"
      }
    }
  }),
  created() {
    if (this.viewMode === 'agent') {
      this.protoAPI.connect().then(
          connection => {
            const date = new Date().toLocaleTimeString();
            this.connection = connection;
            this.connection.subscribe(this.recvData, "data");
            this.$root.NotificationsService.success(`${date} ${this.locale[this.$i18n.locale]['connected']}`);
          },
          error => {
            this.$root.NotificationsService.error(this.locale[this.$i18n.locale]['connError']);
            console.log(error);
          },
      );
    }
  },
  mounted() {
    this.leftTab = this.viewMode === 'agent' ? 'api' : undefined;
  },
  methods: {
    recvData(msg) {
      const date = new Date();
      const date_ms = date.toLocaleTimeString() + `.${date.getMilliseconds()}`;
      this.lines.push(
          `${date_ms} RECV DATA: ${new TextDecoder(
              "utf-8"
          ).decode(msg.content.data)}`
      );
    },
    sendLogLine() {
      const date = new Date();
      const date_ms = date.toLocaleTimeString() + `.${date.getMilliseconds()}`;
      let data = JSON.stringify({ type: "log_line_req", data: "try to send log line", line: this.logline });
      this.lines.push(
          `${date_ms} SEND DATA: ${data}`
      );
      this.connection.sendData(data);
    }
  }
};
</script>
</script>
<style scoped>
.limit-length {
  max-width: 600px;
}
</style>
