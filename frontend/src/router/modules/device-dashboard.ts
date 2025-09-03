const Layout = () => import("@/layout/index.vue");

export default {
  path: "/device-dashboard",
  name: "DeviceDashboard",
  component: Layout,
  redirect: "/device-dashboard/index",
  meta: {
    icon: "ep/grid",
    title: "设备控制台",
    rank: 2
  },
  children: [
    {
      path: "/device-dashboard/index",
      name: "DeviceDashboardIndex",
      component: () => import("@/views/device-dashboard/index.vue"),
      meta: {
        title: "设备控制台",
        showLink: true
      }
    }
  ]
} satisfies RouteConfigsTable;
