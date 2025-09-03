// @ts-check

/** @type {import("prettier").Config} */
export default {
  // 基础配置
  printWidth: 80,                    // 每行最大字符数
  tabWidth: 2,                       // Tab 宽度
  useTabs: false,                    // 使用空格而不是 Tab
  semi: true,                        // 语句末尾添加分号
  singleQuote: false,                // 使用双引号
  quoteProps: "as-needed",           // 对象属性引号：仅在需要时添加
  trailingComma: "none",             // 尾随逗号：不添加
  bracketSpacing: true,              // 对象字面量的大括号间添加空格
  bracketSameLine: false,            // 多行 HTML 元素的 > 放在下一行
  arrowParens: "avoid",              // 箭头函数参数括号：仅在必要时添加
  endOfLine: "lf",                   // 换行符：LF (Unix)

  // Vue 特定配置
  vueIndentScriptAndStyle: false,    // Vue 文件中的 script 和 style 标签不缩进

  // HTML 配置
  htmlWhitespaceSensitivity: "css",  // HTML 空白敏感性

  // 文件覆盖配置
  overrides: [
    {
      files: "*.vue",
      options: {
        parser: "vue"
      }
    },
    {
      files: ["*.ts", "*.tsx"],
      options: {
        parser: "typescript"
      }
    },
    {
      files: ["*.js", "*.jsx"],
      options: {
        parser: "babel"
      }
    },
    {
      files: "*.json",
      options: {
        parser: "json"
      }
    },
    {
      files: "*.css",
      options: {
        parser: "css"
      }
    },
    {
      files: "*.scss",
      options: {
        parser: "scss"
      }
    },
    {
      files: "*.md",
      options: {
        parser: "markdown",
        printWidth: 100,
        proseWrap: "preserve"
      }
    }
  ]
};
