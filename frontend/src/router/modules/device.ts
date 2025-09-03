const Layout = () => import("@/layout/index.vue");

export default {
  path: "/device",
  name: "Device",
  component: Layout,
  redirect: "/device/list",
  meta: {
    icon: "ep/monitor",
    title: "设备管理",
    rank: 1
  },
  children: [
    {
      path: "/device/list",
      name: "DeviceList",
      component: () => import("@/views/device/list/index.vue"),
      meta: {
        title: "设备列表",
        showLink: true
      }
    },
    {
      path: "/device/dashboard",
      name: "DeviceDashboard",
      component: () => import("@/views/device-dashboard/index.vue"),
      meta: {
        title: "设备控制台",
        showLink: false
      }
    },
    {
      path: "/device/detail/:id",
      name: "DeviceDetail",
      component: () => import("@/views/device/detail/index.vue"),
      meta: {
        title: "设备详情",
        showLink: false,
        activePath: "/device/list"
      }
    },
    {
      path: "/device/console/:id",
      name: "DeviceConsole",
      component: () => import("@/views/device/console/index.vue"),
      meta: {
        title: "远程控制台",
        showLink: false,
        activePath: "/device/list"
      }
    }
  ]
} satisfies RouteConfigsTable;
