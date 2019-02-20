package controllers

import (
	"github.com/astaxie/beego"
	"regexp"
	"github.com/astaxie/beego/orm"
	"dailyFresh/models"
	"github.com/astaxie/beego/utils"
	"strconv"
	"encoding/base64"
	//"encoding/base32"
	//"bytes"
	"github.com/gomodule/redigo/redis"
)

type UserController struct {
	beego.Controller
}

//展示注册页面
func(this*UserController)ShowRegister(){
	//指定注册页面
	this.TplName = "register.html"
}

//处理注册业务
func(this*UserController)HandleRegister(){
	//获取数据
	userName := this.GetString("user_name")
	pwd := this.GetString("pwd")
	cpwd := this.GetString("cpwd")
	email := this.GetString("email")
	//校验数据
	if userName == "" || pwd == "" || cpwd == "" || email == ""{
		this.Data["errmsg"] = "输入数据不能为空"
		this.TplName = "register.html"
		return
	}
	if pwd != cpwd{
		this.Data["errmsg"] = "两次密码输入不一致"
		this.TplName = "register.html"
		return
	}

	reg,_ := regexp.Compile(`^[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	result := reg.FindString(email)
	if result == ""{
		this.Data["errmsg"] = "邮箱格式不正确"
		this.TplName = "register.html"
		return
	}

	//处理数据
	o := orm.NewOrm()
	//获取插入对象
	var user models.User
	//给插入对象赋值
	user.UserName = userName
	user.Pwd = pwd
	user.Email = email
	//插入
	o.Insert(&user)

	//发送邮件
	emailConfig := `{"username":"563364657@qq.com","password":"tmauceruuvvzbfec","host":"smtp.qq.com","port":587}`
	emailSend := utils.NewEMail(emailConfig)
	emailSend.From = "563364657@qq.com"
	emailSend.To = []string{email}

	emailSend.Subject = "天天生鲜用户激活"
	emailSend.HTML = `<a href="http://192.168.110.81:8080/active?id=`+strconv.Itoa(user.Id)+`">点击激活用户</a>`

	emailSend.Send()

	//返回数据
	//this.Redirect("/login",302)
	this.Ctx.WriteString("注册成功，请去邮箱激活当前用户")
}

//激活当前用户
func(this*UserController)HandleActive(){
	//获取数据
	id,err :=this.GetInt("id")
	//校验数据
	if err != nil {
		this.Data["errmsg"] = "激活用户失败"
		this.TplName = "register.html"
		return
	}
	//处理数据
	//更新操作
	o := orm.NewOrm()
	//获取一个更新对象
	var user models.User
	//给更新对象赋值
	user.Id = id
	//查询
	err = o.Read(&user)
	//更新
	if err != nil{
		this.Data["errmsg"] = "激活用户失败"
		this.TplName = "register.html"
		return
	}
	user.Active = true
	o.Update(&user)


	//返回数据
	this.Redirect("/login",302)
}

//展示登录页面
func(this*UserController)ShowLogin(){
	//获取cookie信息
	userName := this.Ctx.GetCookie("userName")


	//遗留base64使用
	result ,_:=base64.StdEncoding.DecodeString(userName)
	res := string(result)

	if res == ""{
		this.Data["userName"] = ""
		this.Data["checked"] = ""
	}else {
		this.Data["userName"] = res
		this.Data["checked"] = "checked"
	}

	this.TplName = "login.html"
}

//base64

//处理登录业务
func(this*UserController)HandleLogin(){
	//获取数据
	userName := this.GetString("username")
	pwd :=this.GetString("pwd")
	//校验数据
	if userName == "" || pwd == ""{
		this.Data["errmsg"] = "登录失败"
		this.TplName = "login.html"
		return
	}
	//处理数据
	//查询校验
	o := orm.NewOrm()
	//获取查询对象
	var user models.User
	//给查询条件赋值
	user.UserName = userName
	//查询
	err := o.Read(&user,"UserName")
	if err != nil{
		this.Data["errmsg"] = "用户名不存在"
		this.TplName = "login.html"
		return
	}

	if user.Pwd != pwd{
		this.Data["errmsg"] = "密码错误"
		this.TplName = "login.html"
		return
	}

	if user.Active == false{
		this.Data["errmsg"] = "当前用户未激活,请先去邮箱激活"
		this.TplName = "login.html"
		return
	}

	//登录成功情况下，点击记住用户名
	remember := this.GetString("remember")
	//存入cookie的数据默认不能中文，用base64转字符串

	result :=base64.StdEncoding.EncodeToString([]byte(userName))

	if remember == "on"{
		this.Ctx.SetCookie("userName",result,60 * 60)
	}else{
		this.Ctx.SetCookie("userName",result,-1)
	}

	//记住登录状态
	this.SetSession("userName",userName)

	//返回数据
	this.Redirect("/",302)
}

//退出登录
func(this*UserController)Logout(){
	//清楚session
	this.DelSession("userName")
	//返回页面
	this.Redirect("/login",302)
}

//展示用户中心信息页
func(this*UserController)ShowUserCenterInfo(){

	//获取用户名和默认地址  作业一

	//展示用户浏览记录
	userName := this.GetSession("userName")

	//链接redis获取数据
	conn,err :=redis.Dial("tcp","192.168.110.81:6379")
	if err != nil{
		beego.Error("redis链接失败")
		return
	}
	resp,err :=conn.Do("lrange","history_"+userName.(string),0,4)
	ids,_ :=redis.Ints(resp,err)
	var goods []models.GoodsSKU
	o := orm.NewOrm()
	for _ ,id := range ids{
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)

		goods = append(goods, goodsSku)
	}


	this.Data["goods"] = goods
	this.Layout = "layout.html"
	this.TplName = "user_center_info.html"
}

//展示用户中心订单页
func(this*UserController)ShowUserCenterOrder(){
	//定义一个大容器
	var goods []map[string]interface{}


	//获取数据  获取订单信息表和订单商品表数据
	var orderInfos []models.OrderInfo
	o := orm.NewOrm()
	userName := this.GetSession("userName")
	o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__UserName",userName.(string)).All(&orderInfos)
	//获取订单商品表的内容
	for _,orderInfo := range orderInfos{
		temp := make(map[string]interface{})
		//把订单信息放到行容器
		temp["orderInfo"]  = orderInfo
		//获取订单商品
		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo","GoodsSKU").Filter("OrderInfo",orderInfo).All(&orderGoods)
		temp["orderGoods"] = orderGoods

		goods = append(goods,temp)
	}

	this.Data["goods"] = goods
	//把数据传递到前端



	this.Layout = "layout.html"
	this.TplName = "user_center_order.html"
}

//展示用户中心地址页
func(this*UserController)ShowUserCenterSite(){
	//获取默认地址
	o := orm.NewOrm()
	//获取查询对象
	var address models.Address
	//查询
	userName := this.GetSession("userName")
	qs := o.QueryTable("Address").RelatedSel("User").Filter("User__UserName",userName.(string))
	qs.Filter("Default",true).One(&address)
	this.Data["address"] = address

	this.Layout = "layout.html"
	this.TplName = "user_center_site.html"
}

//处理地址信息
func(this*UserController)HandleSite(){
	//获取数据
	receiver := this.GetString("receiver")
	addr := this.GetString("addr")
	zipCode := this.GetString("zipCode")
	phone := this.GetString("phone")
	//校验数据
	if receiver == "" || addr == "" || zipCode == "" || phone == ""{
		this.Data["errmsg"] = "输入数据不完整，请重新输入"
		this.TplName = "user_center_site.html"
		return
	}
	//邮编格式校验   电话号码格式教研

	//处理数据
	o := orm.NewOrm()
	//获取插入对象
	var address models.Address
	//给插入对象赋值
	address.Receiver = receiver
	address.Addr = addr
	address.ZipCode = zipCode
	address.Phone = phone

	//获取当前用户
	userName := this.GetSession("userName")
	var user models.User
	user.UserName = userName.(string)
	o.Read(&user,"UserName")

	address.Default = true

	address.User = &user

	//查询是否有默认地址，如果有，更新为非默认地址
	var oldAddress models.Address
	qs :=o.QueryTable("Address").RelatedSel("User").Filter("User__UserName",userName.(string))
	err := qs.Filter("Default",true).One(&oldAddress)
	if err == nil{
		oldAddress.Default = false
		o.Update(&oldAddress)
	}

	//插入
	o.Insert(&address)

	//返回数据
	this.Redirect("/goods/userCenterSite",302)
}