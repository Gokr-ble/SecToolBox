<template>
  <n-card :title="`常用工具集锦（当前数量：${tools.length}）`" style="max-width: calc(100vw - 64px); margin: 32px auto; height: calc(100vh - 64px);" content-style="display: flex; flex-direction: column; flex: 1; overflow: hidden; padding: 12px 24px;">
    <div style="display: flex; gap: 16px; flex: 1; min-height: 0;">
      <!-- 左侧类别侧边栏 -->
      <n-card size="small" style="width: 200px; flex-shrink: 0; overflow-y: auto;" content-style="padding: 8px;">
        <n-menu
          v-model:value="activeCategory"
          :options="menuOptions"
          :default-value="activeCategory"
        />
      </n-card>

      <!-- 右侧内容区 -->
      <div style="flex: 1; min-width: 0; display: flex; flex-direction: column; overflow: hidden;">
        <n-space align="center" style="margin-bottom: 12px; flex-shrink: 0;">
          <n-button type="primary" @click="fetchTools">刷新</n-button>
          <n-input v-model:value="filterName" placeholder="按名称筛选" style="width: 180px;" clearable />
          <n-button type="primary" @click="showAddModal = true">新增工具</n-button>
          <n-button type="primary" @click="showConfigEnvModal = true">配置环境变量</n-button>
          <n-button type="primary" @click="openEditCategories">编辑工具类别</n-button>
        </n-space>

        <div style="flex: 1; overflow-y: auto;">
          <n-table :single-line="false" :bordered="true">
          <thead>
            <tr>
              <th>名称</th>
              <th>类型</th>
              <th>路径</th>
              <th>描述</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tool in filteredTools" :key="tool.ID">
              <td>{{ tool.Name }}</td>
              <td style="min-width: 60px">{{ typeMap[tool.Type] || tool.Type }}</td>
              <td style="max-width: 280px;">
                <MiddleEllipsis :text="tool.Path" :max-length="40" />
              </td>
              <td style="max-width: 280px;">
                <MiddleEllipsis :text="tool.Description" :max-length="40" />
              </td>
              <td>
                <div style="display: flex; justify-content: center; align-items: center;">
                  <n-button type="primary" size="small" @click="startTool(tool.ID)">启动</n-button>
                  <n-button type="info" size="small" style="margin-left: 6px;" :disabled="!(tool.Type.endsWith('-cli') || tool.Type === 'python')" @click="openCliTerminal(tool)">终端</n-button>
                  <n-button type="default" size="small" style="margin: 0 12px;"
                    @click="openEditModal(tool)">编辑</n-button>
                  <n-button type="error" size="small" @click="deleteTool(tool.ID)">删除</n-button>
                </div>
              </td>
            </tr>
          </tbody>
          </n-table>
        </div>
      </div>
    </div>

    <!-- 新增工具 Modal -->
    <n-modal v-model:show="showAddModal" title="新增工具">
      <n-card style="max-width: 600px; margin: 0 auto;">
        <n-form :model="newTool" label-width="80" label-placement="left">
          <n-form-item label="名称" path="Name" :feedback="errors.Name"
            :validation-status="errors.Name ? 'error' : undefined">
            <n-input v-model:value="newTool.Name" />
          </n-form-item>
          <n-form-item label="类型" path="Type" :feedback="errors.Type"
            :validation-status="errors.Type ? 'error' : undefined">
            <n-select v-model:value="newTool.Type" :options="typeOptions" />
          </n-form-item>
          <n-form-item label="路径" path="Path" :feedback="errors.Path"
            :validation-status="errors.Path ? 'error' : undefined">
            <n-input v-model:value="newTool.Path" placeholder="如果是路径形式，请以斜线结尾" />
          </n-form-item>
          <n-form-item label="Java版本" path="JavaVersion" :feedback="errors.JavaVersion"
            :validation-status="errors.JavaVersion ? 'error' : undefined">
            <n-input v-model:value="newTool.JavaVersion" placeholder="非Java工具可不填" />
          </n-form-item>
          <n-form-item label="描述" path="Description" :feedback="errors.Description"
            :validation-status="errors.Description ? 'error' : undefined">
            <n-input v-model:value="newTool.Description" />
          </n-form-item>
          <n-form-item label="分类" path="Category" :feedback="errors.Category"
            :validation-status="errors.Category ? 'error' : undefined">
            <n-select v-model:value="newTool.Category" :options="categorySelectOptions" placeholder="请选择分类" />
          </n-form-item>
        </n-form>
        <template #action>
          <n-space justify="center" style="margin-top: 16px;">
            <n-button @click="showAddModal = false">取消</n-button>
            <n-button type="primary" @click="addTool">确定</n-button>
          </n-space>
        </template>
      </n-card>
    </n-modal>

    <!-- 编辑工具 Modal -->
    <n-modal v-model:show="showEditModal" title="编辑工具">
      <n-card style="max-width: 600px; margin: 0 auto;">
        <n-form :model="editTool" label-width="80" label-placement="left">
          <n-form-item label="名称" path="Name" :feedback="editErrors.Name"
            :validation-status="editErrors.Name ? 'error' : undefined">
            <n-input v-model:value="editTool.Name" />
          </n-form-item>
          <n-form-item label="类型" path="Type" :feedback="editErrors.Type"
            :validation-status="editErrors.Type ? 'error' : undefined">
            <n-select v-model:value="editTool.Type" :options="typeOptions" />
          </n-form-item>
          <n-form-item label="路径" path="Path" :feedback="editErrors.Path"
            :validation-status="editErrors.Path ? 'error' : undefined">
            <n-input v-model:value="editTool.Path" />
          </n-form-item>
          <n-form-item label="Java版本" path="JavaVersion" :feedback="editErrors.JavaVersion"
            :validation-status="editErrors.JavaVersion ? 'error' : undefined">
            <n-input v-model:value="editTool.JavaVersion" />
          </n-form-item>
          <n-form-item label="描述" path="Description" :feedback="editErrors.Description"
            :validation-status="editErrors.Description ? 'error' : undefined">
            <n-input v-model:value="editTool.Description" />
          </n-form-item>
          <n-form-item label="分类" path="Category" :feedback="editErrors.Category"
            :validation-status="editErrors.Category ? 'error' : undefined">
            <n-select v-model:value="editTool.Category" :options="categorySelectOptions" placeholder="请选择分类" />
          </n-form-item>
        </n-form>
        <template #action>
          <n-space justify="center" style="margin-top: 16px;">
            <n-button @click="showEditModal = false">取消</n-button>
            <n-button type="primary" @click="saveEditTool">保存</n-button>
          </n-space>
        </template>
      </n-card>
    </n-modal>

    <!-- 配置环境变量 Modal -->
    <n-modal v-model:show="showConfigEnvModal" title="配置环境变量">
      <n-card style="max-width: 800px; margin: 0 auto;">
        <n-form label-width="60" label-placement="left">
          <n-form-item label="Java">
            <n-dynamic-input
              v-model:value="envConfig.java"
              placeholder="暂无Java版本配置"
              :on-create="createJavaVersion"
            >
              <template #default="{ value, index }">
                <n-input-group>
                  <n-input
                    v-model:value="value.version"
                    placeholder="版本号"
                    style="width: 120px;"
                  />
                  <n-input
                    v-model:value="value.path"
                    placeholder="Java可执行文件路径"
                    style="flex: 1;"
                  />
                  <n-button @click="selectJavaFile(index)">浏览</n-button>
                </n-input-group>
              </template>
            </n-dynamic-input>
          </n-form-item>
          <n-form-item label="Python">
            <n-input-group>
              <n-input
                v-model:value="envConfig.python"
                placeholder="Python可执行文件路径"
                style="flex: 1;"
              />
              <n-button @click="selectPythonFile">浏览</n-button>
            </n-input-group>
          </n-form-item>
        </n-form>
        <template #action>
          <n-space justify="center" style="margin-top: 16px;">
            <n-button @click="showConfigEnvModal = false">取消</n-button>
            <n-button type="primary" @click="saveEnvConfig">保存</n-button>
          </n-space>
        </template>
      </n-card>
    </n-modal>

    <!-- 编辑工具类别 Modal -->
    <n-modal v-model:show="showEditCategoriesModal" title="编辑工具类别">
      <n-card style="max-width: 500px; margin: 0 auto;">
        <n-space vertical>
          <n-dynamic-input
            v-model:value="editingCategories"
            placeholder="暂无类别，点击添加"
            :on-create="() => ''"
          >
            <template #default="{ value, index }">
              <n-input-group>
                <n-input v-model:value="editingCategories[index]" placeholder="类别名称" style="flex: 1;" />
              </n-input-group>
            </template>
          </n-dynamic-input>
        </n-space>
        <template #action>
          <n-space justify="center" style="margin-top: 16px;">
            <n-button @click="showEditCategoriesModal = false">取消</n-button>
            <n-button type="primary" @click="saveCategories">保存</n-button>
          </n-space>
        </template>
      </n-card>
    </n-modal>

    <!-- CLI 终端 -->
    <CliTerminal
      v-if="selectedCliTool"
      :tool="selectedCliTool"
      v-model:visible="showCliTerminal"
      :java-versions="envConfig.java"
    />
  </n-card>
</template>

<script lang="ts" setup>
import { ref, onMounted, computed, h } from 'vue'
import type { Ref } from 'vue'
import { GetTools, SaveTools, StartTool, OpenFileDialog, GetEnvConfig, SaveEnvConfig, GetCategories, SaveCategories } from '../../wailsjs/go/main/App'
import { NButton, NCard, NSpace, NTable, NForm, NFormItem, NModal, NInput, NSelect, NMenu, NInputGroup, NDynamicInput, useMessage, useDialog, NTooltip } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import CliTerminal from './CliTerminal.vue'

const MiddleEllipsis = {
  props: {
    text: { type: String, default: '' },
    maxLength: { type: Number, default: 20 }
  },
  setup(props: any) {
    const displayText = computed(() => {
      const text = props.text
      const maxLen = props.maxLength
      if (text.length <= maxLen) return text
      if (text.includes('\\')) {
        const parts = text.split('\\')
        if (parts.length >= 3) {
          if (text.charAt(text.length-1) != '\\') {
            const firstPart = parts[0] + '\\' + parts[1] + '\\'
            const lastPart = '\\' + parts[parts.length - 1]
            const middle = ' ... '
            return firstPart + middle + lastPart
          } else {
            const firstPart = parts[0] + '\\' + parts[1] + '\\'
            const lastPart = '\\' + parts[parts.length - 2] + '\\'
            const middle = ' ... '
            return firstPart + middle + lastPart
          }
        }
      }
      const half = Math.floor((maxLen - 5) / 2)
      return text.slice(0, half) + ' ... ' + text.slice(-half)
    })
    return () => h(NTooltip, { delay: 300 }, {
      trigger: () => h('span', displayText.value),
      default: () => props.text
    })
  }
}

interface ToolConfig {
  ID: string
  Name: string
  Type: string
  Path: string
  JavaVersion: string
  Description: string
  Category: string
}

interface EnvConfig {
  java: Array<{ version: string, path: string }>
  python: string
}

const typeMap: Record<string, string> = {
  'java-gui': 'Java GUI',
  'java-cli': 'Java CLI',
  'python': 'Python',
  'exe-gui': 'EXE GUI',
  'exe-cli': 'EXE CLI'
}
const typeOptions = Object.entries(typeMap).map(([value, label]) => ({ value, label }))

const tools: Ref<ToolConfig[]> = ref([])
const categories: Ref<string[]> = ref([])
const showAddModal = ref(false)
const showEditModal = ref(false)
const showConfigEnvModal = ref(false)
const showEditCategoriesModal = ref(false)
const newTool = ref<Omit<ToolConfig, 'ID'>>({
  Name: '',
  Type: '',
  Path: '',
  JavaVersion: '',
  Description: '',
  Category: '',
})
const editTool = ref<ToolConfig>({
  ID: '',
  Name: '',
  Type: '',
  Path: '',
  JavaVersion: '',
  Description: '',
  Category: '',
})
const envConfig = ref<EnvConfig>({
  java: [],
  python: ''
})
const editingCategories: Ref<string[]> = ref([])

const message = useMessage()
const dialog = useDialog()

const errors = ref<Record<string, string>>({
  Name: '',
  Type: '',
  Path: '',
  JavaVersion: '',
  Description: '',
  Category: ''
})
const editErrors = ref<Record<string, string>>({
  Name: '',
  Type: '',
  Path: '',
  JavaVersion: '',
  Description: '',
  Category: ''
})

const filterName = ref('')
const activeCategory = ref('')
const selectedCliTool = ref<ToolConfig | null>(null)
const showCliTerminal = ref(false)

const menuOptions = computed<MenuOption[]>(() => [
  { label: '全部', key: '' },
  ...categories.value.map(c => ({ label: c, key: c }))
])

const categorySelectOptions = computed(() =>
  categories.value.map(c => ({ value: c, label: c }))
)

const filteredTools = computed(() => {
  return tools.value.filter(tool =>
    (!activeCategory.value || tool.Category === activeCategory.value) &&
    (!filterName.value || tool.Name.includes(filterName.value))
  )
})

function validateTool(tool: typeof newTool.value | ToolConfig, errorsObj: Record<string, string>) {
  let valid = true
  Object.keys(errorsObj).forEach(key => errorsObj[key] = '')
  if (!tool.Name) {
    errorsObj.Name = '名称不能为空'
    valid = false
  }
  if (!tool.Type) {
    errorsObj.Type = '类型不能为空'
    valid = false
  }
  if (!tool.Path) {
    errorsObj.Path = '路径不能为空'
    valid = false
  }
  if (!tool.Description) {
    errorsObj.Description = '描述不能为空'
    valid = false
  }
  if (!tool.Category) {
    errorsObj.Category = '分类不能为空'
    valid = false
  }
  return valid
}

async function fetchTools() {
  try {
    const result = await GetTools()
    tools.value = result || []
  } catch (e) {
    tools.value = []
  }
}

async function fetchCategories() {
  try {
    const result = await GetCategories()
    categories.value = result || []
  } catch (e) {
    categories.value = []
  }
}

async function saveCategories() {
  try {
    const filtered = editingCategories.value.filter(c => c.trim() !== '')
    await SaveCategories(filtered)
    categories.value = filtered
    showEditCategoriesModal.value = false
    message.success('类别保存成功')
  } catch (e) {
    message.error('保存类别失败')
  }
}

function openEditCategories() {
  editingCategories.value = [...categories.value]
  showEditCategoriesModal.value = true
}

async function addTool() {
  if (!validateTool(newTool.value, errors.value)) {
    message.error('请填写所有必填项')
    return
  }
  const uuid = Date.now().toString()
  const tool: ToolConfig = { ID: uuid, ...newTool.value }
  try {
    await SaveTools([...tools.value, tool])
    showAddModal.value = false
    await fetchTools()
    Object.keys(newTool.value).forEach((k) => (newTool.value[k as keyof typeof newTool.value] = ''))
  } catch (e) {
    message.error('新增工具失败')
  }
}

function openEditModal(tool: ToolConfig) {
  editTool.value = { ...tool }
  Object.keys(editErrors.value).forEach(key => editErrors.value[key] = '')
  showEditModal.value = true
}

async function saveEditTool() {
  if (!validateTool(editTool.value, editErrors.value)) {
    message.error('请填写所有必填项')
    return
  }
  try {
    const idx = tools.value.findIndex(t => t.ID === editTool.value.ID)
    if (idx !== -1) {
      const updatedTools = [...tools.value]
      updatedTools[idx] = { ...editTool.value }
      await SaveTools(updatedTools)
      showEditModal.value = false
      await fetchTools()
    }
  } catch (e) {
    message.error('保存失败')
  }
}

async function deleteTool(id: string) {
  dialog.warning({
    title: '确认删除',
    content: '确定要删除该工具吗？此操作不可撤销。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const newList = tools.value.filter(t => t.ID !== id)
        await SaveTools(newList)
        await fetchTools()
      } catch (e) {
        message.error('删除工具失败')
      }
    }
  })
}

function openCliTerminal(tool: ToolConfig) {
  selectedCliTool.value = tool
  showCliTerminal.value = true
}

async function startTool(id: string) {
  try {
    const result = await StartTool(id, false)
    if (typeof result === 'string') {
      message.success(result)
    }
  } catch (e: any) {
    message.error(e?.message || String(e))
    console.log(e?.message || String(e))
    if (String(e).includes("requires elevation")) {
      dialog.warning({
        title: '权限不足',
        content: '启动该工具需要提升权限，是否以管理员身份重新启动？',
        positiveText: '是',
        negativeText: '否',
        onPositiveClick: async () => {
          try {
            const result = await StartTool(id, true)
            if (typeof result === 'string') {
              message.success(result)
            }
          } catch (e: any) {
            message.error(e?.message || String(e))
          }
        }
      })
    }
  }
}

async function fetchEnvConfig() {
  try {
    const result = await GetEnvConfig()
    if (result) {
      envConfig.value.java = result.Java.map((item: Record<string, string>) => {
        const [version, path] = Object.entries(item)[0]
        return { version, path: path as string }
      })
      envConfig.value.python = result.Python || ''
    }
  } catch (e) {
    message.error('获取环境变量配置失败，请手动配置')
    showConfigEnvModal.value = true;
  }
}

function createJavaVersion() {
  return {
    version: '',
    path: ''
  }
}

async function selectJavaFile(index: number) {
  try {
    const result = await OpenFileDialog()
    if (result) {
      envConfig.value.java[index].path = result
    }
  } catch (e) {
    message.error('选择Java文件失败')
  }
}

async function selectPythonFile() {
  try {
    const result = await OpenFileDialog()
    if (result) {
      envConfig.value.python = result
    }
  } catch (e) {
    message.error('选择Python文件失败')
  }
}

async function saveEnvConfig() {
  try {
    const envToSave = {
      Java: envConfig.value.java.map(item => ({ [item.version]: item.path })),
      Python: envConfig.value.python
    }
    await SaveEnvConfig(envToSave)

    message.success('环境变量配置保存成功')
    showConfigEnvModal.value = false
  } catch (e) {
    message.error('保存环境变量配置失败')
  }
}

onMounted(() => {
  fetchTools()
  fetchCategories()
  fetchEnvConfig()
})
</script>

<style scoped>
:deep(.n-menu) {
  height: 100%;
}
</style>
