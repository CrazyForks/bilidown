import { ResJSON } from "../mixin"
import { TaskStatus, VideoFormat } from "../work/type"

export const getActiveTask = async (): Promise<ActiveTask[]> => {
    const res = await fetch('/api/getActiveTask').then(res => res.json()) as ResJSON<ActiveTask[]>
    if (!res.success) throw new Error(res.message)
    return res.data
}

/** 用于刷新任务实时进度 */
type ActiveTask = {
    bvid: string
    cid: number
    /** 分辨率代码 */
    format: VideoFormat
    /** 视频标题 */
    title: string
    /** 视频发布者 */
    owner: string
    /** 视频封面 */
    cover: string
    /** 任务进度 */
    status: TaskStatus
    /** 文件保存到的目录 */
    folder: string
    /** 任务 ID */
    id: number
    /** 音频文件下载进度 */
    audioProgress: number
    /** 视频文件下载进度 */
    videoProgress: number
    /** 音视频合并进度 */
    mergeProgress: number
    /** 视频时长，秒 */
    duration: number
}