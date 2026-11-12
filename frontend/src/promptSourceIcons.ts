const iconsByDomain: Readonly<Record<string, string>> = {
  'prompthub.xin': 'prompthub.png',
  'civitai.com': 'civitai.png',
  'liblib.art': 'liblib.png',
  'jimeng.jianying.com': 'jimeng.png',
  'dreamina.capcut.com': 'jimeng.png',
  'midjourney.com': 'midjourney.svg',
  'xiaohongshu.com': 'xiaohongshu.png',
  'xhslink.com': 'xiaohongshu.png',
  'github.com': 'github.png',
  'huggingface.co': 'huggingface.png',
  'bilibili.com': 'bilibili.png',
  'b23.tv': 'bilibili.png',
  'douyin.com': 'douyin.png',
}

export function promptSourceIcon(hostname: string) {
  const host = hostname.toLowerCase().replace(/\.$/, '')
  const match = Object.entries(iconsByDomain).find(([domain]) => host === domain || host.endsWith('.' + domain))
  return match ? '/icons/sites/' + match[1] : ''
}
