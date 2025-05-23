<template>
    <div class="card rounded-none bg-base-200 h-full overflow-hidden">
        <div class="card-body space-y-4 pt-2 px-4 overflow-y-auto">
            <!-- General Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.name') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>
                <!-- Theme -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-moon-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.theme') }}</h2>
                    </div>
                    <div class="join w-[10rem]">
                        <button v-for="option in prefStore.themeOption" :key="option.value"
                            class="btn btn-sm join-item min-w-[3rem] border-base-300"
                            :class="{
                                'bg-primary text-white': prefStore.general.theme === option.value,
                                'text-base-content hover:text-base-content': prefStore.general.theme !== option.value
                            }"
                            @click="prefStore.general.theme = option.value; prefStore.savePreferences()">
                            <v-icon v-if="option.value === 'light'" name="ri-sun-line" class="h-4 w-4" />
                            <v-icon v-else-if="option.value === 'dark'" name="ri-moon-line" class="h-4 w-4" />
                            <v-icon v-else name="md-hdrauto" class="h-4 w-4" />
                        </button>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Language -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="co-language" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.language') }}</h2>
                    </div>
                    <select class="select select-sm select-bordered w-[10rem]" v-model="prefStore.general.language"
                        @change="onLanguageChange">
                        <option v-for="option in prefStore.langOption" :key="option.value" :value="option.value">
                            {{ option.label }}
                        </option>
                    </select>
                </div>
                <li class="divider-thin"></li>

                <!-- Proxy-->
                <div class="flex flex-col gap-2 p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <v-icon name="ri-global-line" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.general.proxy') }}</h2>
                        </div>
                        <select class="select select-sm select-bordered w-[10rem]" v-model="prefStore.proxy.type"
                            @change="onProxyChange">
                            <option value="none">{{ $t('settings.general.proxy_none') }}</option>
                            <option value="system">{{ $t('settings.general.proxy_system') }}</option>
                            <option value="manual">{{ $t('settings.general.proxy_manual') }}</option>
                        </select>
                    </div>
                    <template v-if="prefStore.proxy.type === 'manual'">
                        <li class="divider-thin"></li>
                        <div class="flex items-center">
                            <div class="flex items-center gap-2 w-32">
                                <v-icon name="ri-global-line" class="h-4 w-4 text-base-content" />
                                <span class="text-sm text-base-content/60">{{ $t('settings.general.proxy_address')
                                    }}</span>
                            </div>
                            <div class="flex items-center gap-2 justify-end flex-1">
                                <input type="text" class="input input-sm input-bordered w-[10rem] text-left"
                                    v-model="proxyAddress" placeholder="http://127.0.0.1:7890"
                                    @change="onProxyChange" />
                            </div>
                        </div>
                    </template>
                </div>
            </ul>

            <!-- Download Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.download') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Directory -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.download_directory') }}</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2"
                            :class="{ 'text-base-content/40': !prefStore.download?.dir }"
                            :title="prefStore.download?.dir">
                            {{ prefStore.download?.dir }}
                        </span>
                        <button class="btn btn-sm btn-ghost btn-square" @click="onSelectDownloadDir">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                        </button>
                    </div>
                </div>
            </ul>

            <!-- Log Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.log') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Log Level -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-information-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.log_level') }}</h2>
                    </div>
                    <select class="select select-sm w-[10rem] text-left border-base-300"
                        v-model="prefStore.logger.level" @change="prefStore.setLoggerConfig()">
                        <option value="debug">Debug</option>
                        <option value="info">Info</option>
                        <option value="warn">Warn</option>
                        <option value="error">Error</option>
                    </select>
                </div>
                <li class="divider-thin"></li>

                <!-- Log Output -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-save-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.log_output') }}</h2>
                    </div>
                    <div class="flex items-center gap-2">
                        <label class="cursor-pointer label">
                            <span class="label-text mr-2">{{ $t('settings.general.log_console') }}</span>
                            <input type="checkbox" class="toggle toggle-sm toggle-primary"
                                v-model="prefStore.logger.enable_console" @change="prefStore.savePreferences()" />
                        </label>
                        <label class="cursor-pointer label">
                            <span class="label-text mr-2">{{ $t('settings.general.log_file') }}</span>
                            <input type="checkbox" class="toggle toggle-sm toggle-primary"
                                v-model="prefStore.logger.enable_file" @change="prefStore.savePreferences()" />
                        </label>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Log Directory -->
                <div v-if="prefStore.logger.enable_file"
                    class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-folder-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.log_dir') }}</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2"
                            :class="{ 'text-base-content/40': !prefStore.logger.directory }"
                            :title="prefStore.logger.directory">
                            {{ prefStore.logger.directory }}
                        </span>
                        <button class="btn btn-sm btn-ghost btn-square" @click="openDirectory(prefStore.logger.directory)">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                        </button>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Log File Settings -->
                <div v-if="prefStore.logger.enable_file"
                    class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-settings-4-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.log_settings') }}</h2>
                    </div>
                    <div class="flex items-center gap-4 justify-end">
                        <div class="flex items-center gap-2">
                            <span class="text-sm text-base-content/60">{{ $t('settings.general.log_max_size') }}</span>
                            <input type="number" class="input input-sm input-bordered w-16 text-center" min="1" max="100"
                                v-model="prefStore.logger.max_size" @change="prefStore.savePreferences()" />
                            <span class="text-sm text-base-content/60">MB</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="text-sm text-base-content/60">{{ $t('settings.general.log_max_age') }}</span>
                            <input type="number" class="input input-sm input-bordered w-16 text-center" min="1" max="365"
                                v-model="prefStore.logger.max_age" @change="prefStore.savePreferences()" />
                            <span class="text-sm text-base-content/60">{{ $t('settings.general.days') }}</span>
                        </div>
                    </div>
                </div>
            </ul>

            <!-- Config and Cache Path Menu-->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.saved_path') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Config Path -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.config_path') }}</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                            {{ prefPath }}
                        </span>
                        <button class="btn btn-sm btn-ghost btn-square" @click="openDirectory(prefPath)">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                        </button>
                    </div>
                </div>

                <!-- Cache Path -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.data_path') }}</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                            {{ taskDbPath }}
                        </span>
                        <button class="btn btn-sm btn-ghost btn-square" @click="openDirectory(taskDbPath)">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                        </button>
                    </div>
                </div>
            </ul>

            <!-- Listend Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.listend') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- WS Listend Info -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-link" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">WebSocket</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2 select-all"
                            :title="wsListendAddress">
                            {{ wsListendAddress }}
                        </span>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- MCP Listend Info -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-link" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">MCP Server</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2 select-all"
                            :title="mcpListendAddress">
                            {{ mcpListendAddress }}
                        </span>
                    </div>
                </div>
            </ul>
        </div>
    </div>
</template>

<script setup>
import { OpenDirectoryDialog, OpenDirectory } from 'wailsjs/go/systems/Service'
import { GetPreferencesPath, GetTaskDbPath } from 'wailsjs/go/api/PathsAPI'
import { computed, ref, watch, onMounted } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'
import { useI18n } from 'vue-i18n'

const prefStore = usePreferencesStore()
const { t, locale } = useI18n()

// proxy address
const proxyAddress = ref('')

// proxy address format validation function
const isValidProxyAddress = (address) => {
    // check if it matches protocol://host:port format
    const regex = /^(http|https|socks5):\/\/[a-zA-Z0-9.-]+:[0-9]+$/
    return regex.test(address)
}

// watch proxy address
watch(() => prefStore.proxy, (newVal) => {
    if (newVal.proxy_address) {
        proxyAddress.value = newVal.proxy_address
    } else {
        proxyAddress.value = ''
    }
}, { immediate: true, deep: true })

const onLanguageChange = async () => {
    await prefStore.savePreferences()
    locale.value = prefStore.currentLanguage
}

const onProxyChange = async () => {
    // if manual mode but no proxy address is provided, do not apply settings
    if (prefStore.proxy.type === 'manual' && !proxyAddress.value.trim()) {
        return
    }

    // if manual mode, check proxy address format
    if (prefStore.proxy.type === 'manual' && !isValidProxyAddress(proxyAddress.value)) {
        $message.error(t('settings.general.proxy_address_err'))
        return
    }

    // update proxy configuration
    if (prefStore.proxy.type === 'manual') {
        // only store complete proxy address
        prefStore.proxy.proxy_address = proxyAddress.value
    }

    // for none and system mode, or manual mode with proxy address, apply settings
    await prefStore.setProxyConfig()
}

const onSelectDownloadDir = async () => {
    const { success, data, msg } = await OpenDirectoryDialog(downloadDir.value)
    if (success && data?.path && data.path.trim() !== '') {
        // update store directory value
        prefStore.download.dir = data.path
        // call new API to send config to backend
        await prefStore.SetDownloadConfig()
    } else if (msg) {
        $message.error(msg)
    }
}

const downloadDir = computed(() => {
    return prefStore.download?.dir || ""
})

const prefPath = ref('')
const taskDbPath = ref('')

const getPreferencesPath = async () => {
    try {
        const response = await GetPreferencesPath()
        if (response.success) {
            prefPath.value = response.data
        } else {
            $message.error(response.msg)
            return null
        }
    } catch (error) {
        console.error('Failed to get preferences path:', error)
        return null
    }
}

const getTaskDbPath = async () => {
    try {
        const response = await GetTaskDbPath()
        if (response.success) {
            taskDbPath.value = response.data
        } else {
            $message.error(response.msg)
            return null
        }
    } catch (error) {
        console.error('Failed to get task db path:', error)
        return null
    }
}

const openDirectory = async (path) => {
    OpenDirectory(path)
}

// Computed properties for listend addresses
const wsListendAddress = computed(() => {
    const wsInfo = prefStore.listendInfo?.ws;
    if (!wsInfo) return '';
    return `${wsInfo.protocol}://${wsInfo.ip}:${wsInfo.port}/${wsInfo.path}`;
});

const mcpListendAddress = computed(() => {
    const mcpInfo = prefStore.listendInfo?.mcp;
    if (!mcpInfo) return '';
    return `${mcpInfo.protocol}://${mcpInfo.ip}:${mcpInfo.port}/${mcpInfo.path}`;
});


onMounted(() => {
    getPreferencesPath()
    getTaskDbPath()
    // Ensure preferences are loaded, you might already have this logic
    // if (!prefStore.loaded) {
    //    prefStore.loadPreferences(); // Assuming you have a method to load preferences
    // }
})
</script>

<style lang="scss" scoped>
.about-app-title {
    font-weight: bold;
    font-size: 18px;
    margin: 5px;
}

.about-link {
    cursor: pointer;

    &:hover {
        text-decoration: underline;
    }
}

.about-logo {
    width: 72px;
    height: 72px;
}

.card {
    @apply shadow-lg;
}

.menu li a {
    @apply hover:bg-base-200;
}

.about-copyright {
    font-size: 12px;
}
</style>
