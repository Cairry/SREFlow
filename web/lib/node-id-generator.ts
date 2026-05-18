/**
 * 节点ID生成器
 * 使用随机6位字母+数字格式：{nodeType}-{random6}
 */

const CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
const ID_LENGTH = 6;

function getRandomChars(): string {
  const array = new Uint32Array(ID_LENGTH);
  crypto.getRandomValues(array);
  return Array.from(array, (x) => CHARS[x % CHARS.length]).join('');
}

export function generateNodeId(nodeType: string): string {
  return `${nodeType}-${getRandomChars()}`;
}
