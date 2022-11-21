<template>
  <div>
    <el-tabs tab-position="left" v-model="leftTab">
      <el-tab-pane
        name="api"
        :label="locale[$i18n.locale]['api']"
        class="layout-fill overflow-hidden"
        v-if="viewMode === 'agent'"
      >
        <div id="exec_actions" class="layout-margin-xl limit-length">
          <el-input placeholder="Please input" v-model="actionData" class="input-with-select">
            <el-select v-model="actionName" slot="prepend" placeholder="Select">
              <el-option v-for="(id, idx) in module.info.actions"
                :label="id"
                :value="id"
                :key="idx"
              ></el-option>
            </el-select>
            <el-button @click="submitAction" slot="append"
            >{{ locale[$i18n.locale]['buttonExecAction'] }}
            </el-button>
          </el-input>
        </div>
        <div class="layout-fill layout-row layout-row-column layout-row-between scrollable">
          <ul>
            <li :key="line" v-for="line in lines">{{ line }}</li>
          </ul>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
const name = "responder";

module.exports = {
  name,
  props: ["protoAPI", "hash", "module", "eventsAPI", "modulesAPI", "components", "viewMode"],
  data: () => ({
    leftTab: undefined,
    connection: {},
    actionData: '{"key1": "val1"}',
    actionName: "",
    lines: [],
    locale: {
      ru: {
        api: "VX API",
        buttonExecAction: "Выполнить действие",
        connected: "подключен",
        connError: "Ошибка подключения к серверу",
        recvError: "Ошибка при выполнении",
        checkError: "Ошибка при проверке данных",
        actionError: "Выберите действие из списка"
      },
      en: {
        api: "VX API",
        buttonExecAction: "Exec action",
        connected: "connected",
        connError: "Error connection to the server",
        recvError: "Error on execute",
        checkError: "Error on validating action data",
        actionError: "Please choose action from list"
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
    checkActionData() {
      if (!this.actionData || typeof(this.actionData) !== "string"){
        return false;
      }
      try {
        const actionData = JSON.parse(this.actionData);
        if (typeof(actionData) !== "object") {
          return false;
        }
        return true;
      } catch (e) {
        return false;
      }
    },
    submitAction() {
      const date = new Date();
      const date_ms = date.toLocaleTimeString() + `.${date.getMilliseconds()}`;
      if (this.actionName === "") {
        this.$root.NotificationsService.error(this.locale[this.$i18n.locale]['actionError']);
        return;
      }
      if (!this.checkActionData()) {
        this.$root.NotificationsService.error(this.locale[this.$i18n.locale]['checkError']);
        return;
      }
      let data = JSON.stringify({
        data: JSON.parse(this.actionData),
        actions: [`${this.module.info.name}.${this.actionName}`]
      });
      this.lines.push(
          `${date_ms} SEND ACTION: ${data}`
      );
      this.connection.sendAction(data, this.actionName);
    }
  }
};
</script>

<style scoped>
  #exec_actions .el-select .el-input {
    width: 170px;
  }
  .input-with-select .el-input-group__prepend {
    background-color: #fff;
  }
</style>
