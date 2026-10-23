export type CountdownTimer = {
  /** 启动（重复调用会先清理旧 interval）。 */
  start: () => void
  /** 停止并置空；卸载时调用即可确保没有残留定时器。 */
  stop: () => void
  isActive: () => boolean
}

/** 可测试的倒计时定时器，避免组件卸载后 interval 继续运行。 */
export const createCountdownTimer = (onTick: () => void, intervalMs = 1000): CountdownTimer => {
  let timer: ReturnType<typeof setInterval> | null = null

  const stop = () => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  return {
    start: () => {
      stop()
      timer = setInterval(onTick, intervalMs)
    },
    stop,
    isActive: () => timer !== null,
  }
}