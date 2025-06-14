<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { InstallYTDLP, CheckYTDLPUpdate, UpdateYTDLP } from 'wailsjs/go/api/DowntasksAPI'
import { GetYTDLPPath, GetFFMPEGPath, SetFFMpegExecPath } from 'wailsjs/go/api/PathsAPI'
import { useDtStore } from '@/handlers/downtasks'
import { Info, OpenDirectory } from 'wailsjs/go/systems/Service'
import { useI18n } from 'vue-i18n'
import usePreferencesStore from '@/stores/preferences'

const { t } = useI18n()
const prefStore = usePreferencesStore()

const ytdlpStatus = async () => {
    try {
        const response = await GetYTDLPPath()
        if (response.success) {
            const data = JSON.parse(response.data)
            if (data.available) {
                prefStore.dependencies.ytdlp.installed = true
                prefStore.dependencies.ytdlp.path = data.path
                prefStore.dependencies.ytdlp.execPath = data.execPath
                prefStore.dependencies.ytdlp.version = data.version
                // check update
                await checkYTDLPUpdate()
            } else {
                prefStore.dependencies.ytdlp.installed = false
            }
        } else {
            prefStore.dependencies.ytdlp.installed = false
            $message.warning(response.msg)
        }
    } catch (error) {
        prefStore.dependencies.ytdlp.installed = false
        console.error('Get ytdlp status failed:', error)
    }
}

const installYtdlp = async () => {
    if (prefStore.dependencies.ytdlp.installing) return
    prefStore.dependencies.ytdlp.installing = true
    prefStore.dependencies.ytdlp.installProgress = ''

    try {
        const response = await InstallYTDLP()
        if (response.success) {
            await ytdlpStatus()
            $message.success(t('settings.dependency.install_success'))
        } else {
            $message.error(response.msg)
        }
    } catch (error) {
        console.error('Install ytdlp failed:', error)
        $message.error(t('settings.dependency.install_failed'))
    } finally {
        prefStore.dependencies.ytdlp.installing = false
    }
}

const updateYtdlp = async () => {
    if (prefStore.dependencies.ytdlp.updating) return
    prefStore.dependencies.ytdlp.updating = true
    prefStore.dependencies.ytdlp.updateProgress = ''
    try {
        const response = await UpdateYTDLP()
        if (response.success) {
            await ytdlpStatus()
            $message.success(t('settings.dependency.update_success'))
        } else {
            $message.error(response.msg)
        }
    } catch (error) {
        console.error('Update ytdlp failed:', error)
        $message.error(t('settings.dependency.update_failed'))
    } finally {
        prefStore.dependencies.ytdlp.updating = false
    }
}

const ffmpegStatus = async () => {
    try {
        const response = await GetFFMPEGPath()
        if (response.success) {
            const data = JSON.parse(response.data)
            if (data.available) {
                prefStore.dependencies.ffmpeg.installed = true
                prefStore.dependencies.ffmpeg.path = data.path
            } else {
                prefStore.dependencies.ffmpeg.installed = false
            }
        } else {
            prefStore.dependencies.ffmpeg.installed = false
            $message.warning(response.msg)
        }
    } catch (error) {
        prefStore.dependencies.ffmpeg.installed = false
        $message.error('Get ffmpeg status failed:', error)
    }
}

const installFFMPEG = async () => {
    $dialog.info({
        title: t('settings.dependency.install_ffmpeg'),
        content: t('settings.dependency.install_ffmpeg_desc')
    })
}

const openDirectory = async (path) => {
    OpenDirectory(path)
}

const ffmpegExecPath = ref('')
const inputFFMPEGPath = async () => {
    try {
        const response = await SetFFMpegExecPath(ffmpegExecPath.value)
        if (response.success) {
            const data = JSON.parse(response.data)
            if (data.available) {
                prefStore.dependencies.ffmpeg.installed = true
                prefStore.dependencies.ffmpeg.available = data.available
                prefStore.dependencies.ffmpeg.path = data.path
                prefStore.dependencies.ffmpeg.execPath = data.execPath
                $message.success("Set FFMpeg success")
            } else {
                prefStore.dependencies.ffmpeg.installed = false
                $message.error("Set FFMpeg failed, message:", data.msg)
            }
        } else {
            prefStore.dependencies.ffmpeg.installed = false
            $message.warning(response.msg)
        }
    } catch (error) {
        prefStore.dependencies.ffmpeg.installed = false
        $message.error('Set FFMpeg failed, catched error:', error)
    }

}


const currentOS = ref('')
const getOS = async () => {
    const response = await Info()
    if (response.success) {
        currentOS.value = response.data.os
    } else {
        $message.warning(response.msg)
    }

}

const ytdlpNeedUpdate = ref(false)
const checkYTDLPUpdate = async () => {
    const response = await CheckYTDLPUpdate()
    if (response.success) {
        const data = JSON.parse(response.data)
        // update preferences
        prefStore.dependencies.ytdlp.updated = true
        prefStore.dependencies.ytdlp.latestVersion = data.latestVersion
        prefStore.dependencies.ytdlp.needUpdate = data.needUpdate
        // update ytdlpNeedUpdate
        if (data.needUpdate) {
            ytdlpNeedUpdate.value = true
        } else {
            ytdlpNeedUpdate.value = false
        }
    } else {
        $message.warning(response.msg)
    }
}

const handleInstalling = (progress) => {
    if (progress.stage == "installing") {
        prefStore.dependencies.ytdlp.installProgress = progress.percentage.toFixed(2) + '%'
    } else if (progress.stage == "installed") {
        prefStore.dependencies.ytdlp.installProgress = ''
        ytdlpStatus()
    } else if (progress.stage == "updating") {
        prefStore.dependencies.ytdlp.updateProgress = progress.percentage.toFixed(2) + '%'
    } else if (progress.stage == "updated") {
        prefStore.dependencies.ytdlp.updateProgress = ''
        ytdlpStatus()   
    } else {
        prefStore.dependencies.ytdlp.installProgress = ''
        $message.error("unknown stage: " + progress.stage)
    }
}

// lifecycle hooks
onMounted(() => {
    // get current os
    getOS()
    // get ytdlp status
    ytdlpStatus()
    // get ffmpeg status
    ffmpegStatus()
    // check ytdlp update
    checkYTDLPUpdate()

    // ws
    const dtStore = useDtStore()
    dtStore.registerInstallingCallback(handleInstalling)

    // cleanup function
    onUnmounted(() => {
        dtStore.unregisterInstallingCallback(handleInstalling)
    })
})

</script>

<template>
    <div class="card rounded-none bg-base-200">
        <div class="card-body space-y-4 pt-2 px-4">
            <!-- Links Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <h2 class="font-semibold text-base-content">{{ $t('settings.dependency.ytdlp') }}</h2>
                    <div v-if="prefStore.dependencies.ytdlp.installed">
                        <button @click="ytdlpStatus()"
                            class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                $t('settings.dependency.refresh') }}</button>
                    </div>
                    <div v-else>
                        <div class="flex items-center gap-2 justify-end">
                            <span v-if="prefStore.dependencies.ytdlp.installProgress"
                                class="text-sm text-base-content/60">
                                {{ prefStore.dependencies.ytdlp.installProgress }}
                            </span>
                            <button @click="installYtdlp()" :disabled="prefStore.dependencies.ytdlp.installing"
                                class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                    $t('settings.dependency.install') }}</button>
                        </div>
                    </div>
                </div>
                <div v-if="prefStore.dependencies.ytdlp.installed">
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.path') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ytdlp.path }}
                            </span>
                            <button class="btn btn-sm btn-ghost btn-square"
                                @click="openDirectory(prefStore.dependencies.ytdlp.path)">
                                <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                            </button>
                        </div>
                    </div>
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-versions" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.version') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ytdlp.version }}
                            </span>
                        </div>
                    </div>
                    <div v-if="prefStore.dependencies.ytdlp.needUpdate">
                        <li class="divider-thin"></li>
                        <div v-if="prefStore.dependencies.ytdlp.updating">
                            <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                                <div class="flex items-center gap-2">
                                    <v-icon name="oi-versions" class="h-4 w-4 text-base-content" />
                                    <h2 class="text-base-content">{{ $t('settings.dependency.updating') }}</h2>
                                </div>
                                <div class="join items-center">
                                    <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                        {{ prefStore.dependencies.ytdlp.latestVersion }}
                                    </span>
                                    <span v-if="prefStore.dependencies.ytdlp.updateProgress"
                                        class="text-sm text-base-content/60">
                                        {{ prefStore.dependencies.ytdlp.updateProgress }}
                                    </span>
                                </div>
                            </div>
                        </div>
                        <div v-else class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                            <div class="flex items-center gap-2">
                                <v-icon name="oi-versions" class="h-4 w-4 text-base-content" />
                                <h2 class="text-base-content">{{ $t('settings.dependency.latest_version') }}</h2>
                            </div>
                            <div class="join items-center">
                                <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                    {{ prefStore.dependencies.ytdlp.latestVersion }}
                                </span>
                                <button class="btn btn-sm border-1 border-base-300 font-normal text-base-content"
                                    @click="updateYtdlp()">
                                    {{ $t('settings.dependency.update') }}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </ul>

            <!-- ffmpeg -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <h2 class="font-semibold text-base-content">{{ $t('settings.dependency.ffmpeg') }}</h2>
                    <div v-if="prefStore.dependencies.ffmpeg.installed">
                        <button @click="ffmpegStatus()"
                            class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                $t('settings.dependency.refresh') }}</button>
                    </div>
                    <div v-else>
                        <div class="flex items-center gap-2 justify-end">
                            <button @click="ffmpegStatus()"
                                class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                    $t('settings.dependency.refresh') }}</button>
                            <button @click="installFFMPEG()"
                                class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                    $t('settings.dependency.install') }}</button>
                        </div>
                    </div>
                </div>
                <div v-if="prefStore.dependencies.ffmpeg.installed">
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.path') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ffmpeg.path }}
                            </span>
                            <button class="btn btn-sm btn-ghost btn-square"
                                @click="openDirectory(prefStore.dependencies.ffmpeg.path)">
                                <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                            </button>
                        </div>
                    </div>
                </div>
                <div v-else>
                    <div v-if="currentOS == 'darwin'">
                        <li class="divider-thin"></li>
                        <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                            <div class="flex items-center gap-2">
                                <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                                <h2 class="text-base-content">{{ $t('settings.dependency.path') }}</h2>
                            </div>
                            <div class="join">
                                <div>
                                    <label class="input join-item input-sm">
                                        <input type="text" placeholder="/usr/local/bin/ffmpeg" v-model="ffmpegExecPath"
                                            required />
                                    </label>
                                </div>
                                <button class="btn btn-neutral btn-sm join-item" @click="inputFFMPEGPath()">{{
                                    $t('settings.dependency.check') }}</button>
                            </div>
                        </div>
                    </div>
                </div>
            </ul>
        </div>
    </div>
</template>

<style lang="scss" scoped>
</style>
