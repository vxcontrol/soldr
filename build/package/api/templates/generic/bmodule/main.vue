<template>
  <div>
    <el-tabs tab-position="left" v-model="leftTab">
      <el-tab-pane name="api" :label="locale[$i18n.locale]['api']" v-if="viewMode === 'agent'">
        <p class="layout-margin-xl buttons">
          <el-button @click="submitData"
          >{{ locale[$i18n.locale]['buttonData'] }}
          </el-button>
          <el-button @click="submitText"
          >{{ locale[$i18n.locale]['buttonText'] }}
          </el-button>
        </p>
        <ul>
          <li :key="line" v-for="line in lines">{{ line }}</li>
        </ul>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
const name = "generic";

module.exports = {
  name,
  props: ["protoAPI", "hash", "module", "eventsAPI", "modulesAPI", "components", "viewMode"],
  data: () => ({
    leftTab: undefined,
    connection: {},
    lines: [],
    locale: {
      ru: {
        api: "VX API",
        buttonData: "Отправить data",
        buttonText: "Отправить text",
        connected: "подключен",
        connError: "Ошибка подключения к серверу",
        recvError: "Ошибка при выполнении"
      },
      en: {
        api: "VX API",
        buttonData: "Send data",
        buttonText: "Send text",
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
    submitData() {
      const date = new Date();
      const date_ms = date.toLocaleTimeString() + `.${date.getMilliseconds()}`;
      let data = JSON.stringify({ type: "hs_browser", data: "test test test" });
      this.lines.push(
          `${date_ms} SEND DATA: ${data}`
      );
      this.connection.sendData(data);
    },
    submitText() {
      const date = new Date();
      const date_ms = date.toLocaleTimeString() + `.${date.getMilliseconds()}`;
      let text = "simple request";
      this.lines.push(
          `${date_ms} SEND TEXT: ${text}`
      );
      this.connection.sendText(text);
    }
  }
};
</script>
