package routers

import (
	"dailyFresh/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
   // beego.Router("/", &controllers.MainController{})
   	beego.InsertFilter("/goods/*",beego.BeforeExec,filterFunc)
    beego.Router("/register",&controllers.UserController{},"get:ShowRegister;post:HandleRegister")
    //激活用户
    beego.Router("/active",&controllers.UserController{},"get:HandleActive")
    //登录业务
    beego.Router("/login",&controllers.UserController{},"get:ShowLogin;post:HandleLogin")
    //首页
    beego.Router("/",&controllers.GoodsController{},"get:ShowIndex")
    //退出登录
    beego.Router("/logout",&controllers.UserController{},"get:Logout")
    //用户中心信息页
    beego.Router("/goods/userCenterInfo",&controllers.UserController{},"get:ShowUserCenterInfo")
    //用户中心订单页
    beego.Router("/goods/userCenterOrder",&controllers.UserController{},"get:ShowUserCenterOrder")
    //用户中心地址页
    beego.Router("/goods/userCenterSite",&controllers.UserController{},"get:ShowUserCenterSite;post:HandleSite")
    //商品详情
    beego.Router("/goodsDetail",&controllers.GoodsController{},"get:ShowDetail")
    //商品列表页
    beego.Router("/goodsList",&controllers.GoodsController{},"get:ShowList")
    //商品搜索
    beego.Router("/search",&controllers.GoodsController{},"post:HandleSearch")
    //添加购物车
    beego.Router("/addCart",&controllers.CartController{},"post:HandleAddCart")
    //展示购物车页面
    beego.Router("/goods/cart",&controllers.CartController{},"get:ShowCart")
    //添加购物车商品数量
    beego.Router("/addCartGoods",&controllers.CartController{},"post:HandleAddCartGoods")
    //删除购物车商品
    beego.Router("/deleteCartGoods",&controllers.CartController{},"post:DeleteCartGoods")
    //展示订单页
    beego.Router("/goods/showOrder",&controllers.OrderController{},"post:HandleShowOrder")
    //添加订单数据
    beego.Router("/addOrder",&controllers.OrderController{},"post:HandleAddOrder")
    //支付宝支付
    beego.Router("/aliPay",&controllers.OrderController{},"get:SendAliPay")
    //支付成功处理
    beego.Router("/payOK",&controllers.OrderController{},"get:HandleAli")
    //发送短信
    beego.Router("/sendMsg",&controllers.OrderController{},"get:SendMsg")
}

func filterFunc(ctx*context.Context){
	userName := ctx.Input.Session("userName")
	if userName == nil{
		ctx.Redirect(302,"/login")
		return
	}
}
